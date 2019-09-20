package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// The Job structure holds all of the specified modifiable simulation parameters
type Job struct {
	ID       string  `json:"string"`
	Folder   string  `json:"folder"`
	SLR      float64 `json:"slr"`
	Protection int	`json:"protection"`
	Tide     int     `json:"tide"`
	Analysis int     `json:"analysis"`
}

// toString takes the job's parameters and generates a unique string to reference it with
func (j *Job) toString() string {
	retVal := fmt.Sprintf("__slr_%d", int(j.SLR*10))

	switch j.Tide {
	case 0:
		retVal += "__tide_low"
	case 1:
		retVal += "__tide_zero"
	case 2:
		retVal += "__tide_high"
	}

	switch j.Analysis {
	case 0:
		retVal += "__analysis_deterministic"
	case 1:
		retVal += "__analysis_expected"
	case 2:
		retVal += "__analysis_extreme"
	}

	switch j.Protection {
	case 1:
		retVal += "__protection_current"
	case 2:
		retVal += "__protection_degraded"
	case 3:
		retVal += "__protection_compromised"
	}

	return retVal
}

// The Parameters structure matches the on-disk JSON input file format for model execution
type Parameters struct {
	IndexSLT     [2]int     `json:"index_SLT"`
	IndexW       int        `json:"index_W"`
	IndexProb    float64    `json:"index_prob"`
	Parameters   [6]float64 `json:"param"`
	TimeMC       float64    `json:"timeMC"`
	Latitude     []float64  `json:"lat_track"`
	Longitude    []float64  `json:"long_track"`
	SeaLevelRize float64    `json:"SLR"`
	Tide         float64    `json:"tide"`
	Protection   int        `json:"protection"`
	Strength     int        `json:"ne_strength"`
	StormType    int        `json:"indicator"`
	SurgeOut     string     `json:"surge_file"`
	WindOut      string     `json:"wind_file"`
	RunupOut     string     `json:"runup_file"`
	WorkspaceOut string     `json:"workspace_file"`
}

var currentJob *Job
var requestedJob = false

func main() {
	u, err := url.Parse(fmt.Sprintf("wss://%s/queue?id=%s", os.Getenv("SERVER_HOSTNAME"), os.Getenv("POD_NAME")))
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("Connecting to", u.String())

	h := make(http.Header)
	if len(os.Getenv("HTTP_USER")) > 0 && len(os.Getenv("HTTP_PASS")) > 0 {
		h.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(os.Getenv("HTTP_USER")+":"+os.Getenv("HTTP_PASS"))))
	}

	dialer := websocket.Dialer{}
	c, _, err := dialer.Dial(u.String(), h)
	if err != nil {
		log.Println("Secure connection failed. Attempting insecure.")
		log.Println("Connecting to", strings.Replace(u.String(), "wss", "ws", -1))
		c, _, err = dialer.Dial(strings.Replace(u.String(), "wss", "ws", -1), h)
		if err != nil {
			log.Fatal("dial:", err)
		}
	}
	defer c.Close()

	log.Println("Connection Successful")

	c.SetReadLimit(2048)
	c.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.SetPongHandler(func(string) error {
		c.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	Send := make(chan string)
	// Create Write Loop
	go func(con *websocket.Conn, send chan string) {
		ticker := time.NewTicker(30 * time.Second)
		defer func() {
			ticker.Stop()
			con.Close()
		}()

		for {
			select {
			case data := <-send:
				con.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := con.WriteMessage(websocket.TextMessage, []byte(data)); err != nil {
					return
				}
			case <-ticker.C:
				con.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := con.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
					return
				}
			}
		}
	}(c, Send)

	// Create Read Loop
	go func(con *websocket.Conn) {
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Fatalln("read:", err)
				break
			}

			// No work items remaining
			if string(message) == "DATA:" {
				requestedJob = false
				continue
			}

			// Decode Job
			log.Println("Decoding Job")
			var job Job
			if err := json.Unmarshal([]byte(string(message)[5:]), &job); err != nil {
				log.Fatalln(err)
			}
			currentJob = &job
		}
	}(c)

	for {
		time.Sleep(5 * time.Second)

		if currentJob == nil && !requestedJob {
			Send <- "GET:"
			requestedJob = true
		}

		if currentJob != nil {
			requestedJob = false

			// Execute Job
			log.Println("Executing Job:", currentJob.ID)

			// Download input
			log.Println("Downloading Input")
			var params Parameters
			if currentJob.Tide == -1 && currentJob.Analysis == -1 {
				if err := exec.Command("aws", "s3", "cp", "s3://simulation.njcoast.us/"+currentJob.Folder+"/input_params.json", "/app/input_params.json").Run(); err != nil {
					log.Fatalln(err)
				}

				// Read parameters
				log.Println("Updating Parameters")
				fIn, err := os.Open("/app/input_params.json")
				if err != nil {
					log.Fatalln(err)
				}

				if err := json.NewDecoder(fIn).Decode(&params); err != nil {
					log.Fatalln(err)
				}

				fIn.Close()
			} else {
				if err := exec.Command("aws", "s3", "cp", "s3://simulation.njcoast.us/"+currentJob.Folder+"/input.geojson", "/app/sandy.geojson").Run(); err != nil {
					log.Fatalln(err)
				}

				// Run initial model
				log.Println("Generating Track")
				iCommand := exec.Command("/app/run_ObtainingParametersCrossingPoint.sh", "/opt/matlab/runtime")
				iCommand.Dir = "/app"
				iOutput, err := iCommand.CombinedOutput()
				if err != nil {
					Send <- "FAILED:"
					log.Println(string(iOutput), err)
					currentJob = nil
					continue
				}

				// Read parameters
				log.Println("Updating Parameters")
				fIn, err := os.Open("/app/input_params.json")
				if err != nil {
					log.Fatalln(err)
				}

				if err := json.NewDecoder(fIn).Decode(&params); err != nil {
					log.Fatalln(err)
				}

				fIn.Close()

				// Modify Parameters
				params.SeaLevelRize = currentJob.SLR
				params.Tide = 0.5 * float64(currentJob.Tide)
				params.Protection = currentJob.Protection

				switch currentJob.Analysis {
				case 0:
					params.IndexProb = 0.0
					break
				case 1:
					params.IndexProb = 0.5
					break
				case 2:
					params.IndexProb = 0.1
					break
				}

				fOut, err := os.OpenFile("/app/input_params.json", os.O_WRONLY|os.O_TRUNC, 0644)
				if err != nil {
					log.Fatalln(err)
				}

				if err := json.NewEncoder(fOut).Encode(&params); err != nil {
					log.Fatalln(err)
				}

				fOut.Close()

				// Upload Parameters
				log.Println("Uploading Parameters")
				if err := exec.Command("aws", "s3", "cp", "--acl", "public-read", "/app/input_params.json", "s3://simulation.njcoast.us/"+currentJob.Folder+"/input_params"+currentJob.toString()+".json").Run(); err != nil {
					log.Fatalln(err)
				}

				if err := exec.Command("aws", "s3", "cp", "--acl", "public-read", "/app/cone.json", "s3://simulation.njcoast.us/"+currentJob.Folder+"/cone.json").Run(); err != nil {
					log.Fatalln(err)
				}
			}

			// Run final model
			log.Println("Executing Model")
			fCommand := exec.Command("/app/run_WebCentralAnalysis.sh", "/opt/matlab/runtime")
			fCommand.Dir = "/app"
			fOutput, err := fCommand.CombinedOutput()
			if err != nil {
				Send <- "FAILED:"
				log.Println(string(fOutput), err)
				currentJob = nil
				continue
			}

			convCommand := exec.Command("node", "/root/convert.js")
			convCommand.Dir = "/app"
			convCommand.Run()

			// Upload Result
			log.Println("Upload Results")
			if currentJob.Tide == -1 && currentJob.Analysis == -1 {
				if err := exec.Command("aws", "s3", "cp", "--acl", "public-read", "/app/heatmap.json", "s3://simulation.njcoast.us/"+currentJob.Folder+"/heatmap.json").Run(); err != nil {
					log.Fatalln(err)
				}

				if err := exec.Command("aws", "s3", "cp", "--acl", "public-read", "/app/wind_heatmap.json", "s3://simulation.njcoast.us/"+currentJob.Folder+"/wind_heatmap.json").Run(); err != nil {
					log.Fatalln(err)
				}
				exec.Command("aws", "s3", "cp", "--acl", "public-read", "/app/wind.geojson", "s3://simulation.njcoast.us/"+currentJob.Folder+"/wind.geojson").Run()

				if err := exec.Command("aws", "s3", "cp", "--acl", "public-read", "/app/surge_line.json", "s3://simulation.njcoast.us/"+currentJob.Folder+"/surge_line.json").Run(); err != nil {
					log.Fatalln(err)
				}
				exec.Command("aws", "s3", "cp", "--acl", "public-read", "/app/surge.geojson", "s3://simulation.njcoast.us/"+currentJob.Folder+"/surge.geojson").Run()
				exec.Command("aws", "s3", "cp", "--acl", "public-read", "/app/track.json", "s3://simulation.njcoast.us/"+currentJob.Folder+"/track.geojson").Run()

				if params.StormType == 1 {
					if err := exec.Command("aws", "s3", "cp", "--acl", "public-read", "/app/transect_line.json", "s3://simulation.njcoast.us/"+currentJob.Folder+"/transect_line.json").Run(); err != nil {
						log.Fatalln(err)
					}
				}
			} else {
				if err := exec.Command("aws", "s3", "cp", "--acl", "public-read", "/app/heatmap.json", "s3://simulation.njcoast.us/"+currentJob.Folder+"/heatmap"+currentJob.toString()+".json").Run(); err != nil {
					log.Fatalln(err)
				}

				if err := exec.Command("aws", "s3", "cp", "--acl", "public-read", "/app/wind_heatmap.json", "s3://simulation.njcoast.us/"+currentJob.Folder+"/wind_heatmap"+currentJob.toString()+".json").Run(); err != nil {
					log.Fatalln(err)
				}
				exec.Command("aws", "s3", "cp", "--acl", "public-read", "/app/wind.geojson", "s3://simulation.njcoast.us/"+currentJob.Folder+"/wind"+currentJob.toString()+".geojson").Run()

				if err := exec.Command("aws", "s3", "cp", "--acl", "public-read", "/app/surge_line.json", "s3://simulation.njcoast.us/"+currentJob.Folder+"/surge_line"+currentJob.toString()+".json").Run(); err != nil {
					log.Fatalln(err)
				}
				exec.Command("aws", "s3", "cp", "--acl", "public-read", "/app/surge.geojson", "s3://simulation.njcoast.us/"+currentJob.Folder+"/surge"+currentJob.toString()+".geojson").Run()
				exec.Command("aws", "s3", "cp", "--acl", "public-read", "/app/track.json", "s3://simulation.njcoast.us/"+currentJob.Folder+"/track"+currentJob.toString()+".geojson").Run()

				if params.StormType == 1 {
					if err := exec.Command("aws", "s3", "cp", "--acl", "public-read", "/app/transect_line.json", "s3://simulation.njcoast.us/"+currentJob.Folder+"/transect_line"+currentJob.toString()+".json").Run(); err != nil {
						log.Fatalln(err)
					}
				}
			}

			// Return Success
			Send <- "COMPLETE:" + currentJob.ID
			log.Println("Job Complete")
			currentJob = nil
		}
	}
}
