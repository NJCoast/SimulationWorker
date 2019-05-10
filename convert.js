var unzipper = require('unzipper');
var tj = require('@mapbox/togeojson'), fs = require('fs'), DOMParser = require('xmldom').DOMParser;

if( fs.existsSync('surge.kmz') ){
    fs.createReadStream('surge.kmz')
        .pipe(unzipper.Parse())
        .on('entry', async entry => {
            if( entry.path == "surge.kml" ){
                const content = await entry.buffer();
                var dom = new DOMParser().parseFromString(content.toString('utf8'));
                fs.writeFile('surge.geojson', JSON.stringify(tj.kml(dom, { styles: true })), () => {});
            }else{
                entry.autodrain()
            }
        })
}

if (fs.existsSync('wind.kmz')) {
    fs.createReadStream('wind.kmz')
        .pipe(unzipper.Parse())
        .on('entry', async entry => {
            if( entry.path == "wind.kml" ){
                const content = await entry.buffer();
                var dom = new DOMParser().parseFromString(content.toString('utf8'));
                fs.writeFile('wind.geojson', JSON.stringify(tj.kml(dom, { styles: true })), () => {});
            }else{
                entry.autodrain()
            }
        })
}