#!/bin/bash

# Run Trajectory Calculation
aws s3 cp s3://simulation.njcoast.us/$1/input.geojson sandy.geojson
/app/run_ObtainingParametersCrossingPoint.sh /opt/matlab/runtime
aws s3 cp --acl public-read /app/input_params.json s3://simulation.njcoast.us/$1/input_params.json
aws s3 cp --acl public-read /app/cone.json s3://simulation.njcoast.us/$1/cone.json

# Run Wind Surge Analysis
/app/run_WebCentralAnalysis.sh /opt/matlab/runtime
aws s3 cp --acl public-read /app/heatmap.json s3://simulation.njcoast.us/$1/heatmap.json
aws s3 cp --acl public-read /app/wind_heatmap.json s3://simulation.njcoast.us/$1/wind_heatmap.json
aws s3 cp --acl public-read /app/surge_line.json s3://simulation.njcoast.us/$1/surge_line.json
aws s3 cp --acl public-read /app/transect_line.json s3://simulation.njcoast.us/$1/transect_line.json