# torrentfile
Simple abstraction layer for encoding/decoding .torrent files 

## Installation
```bash
$ go get github.com/fuchsi/torrentfile
```

## Usage
### Encoding
torrentfile.EncodeTorrentFile takes an TorrentFile as argument and returns a []byte.  
Example:
```go
package main

import (
	"fmt"
	
	"github.com/fuchsi/torrentfile"
)

func main() {
	tfile := torrentfile.TorrentFile{}
	tfile.Name = "some torrent"
	tfile.AnnounceUrl = "http://localhost/announce"
	
	fmt.Printf("encoded torrent file: %s\n", torrentfile.EncodeTorrentFile(tfile))
	// or
	fmt.Printf("encoded torrent file: %s\n", tfile.Encode())
}
```

### Decoding
torrentfile.DecodeTorrentFile takes an io.Reader as argument and returns (TorrentFile, error)  
Example:
```go
package main

import (
	"fmt"
	"log"
	"os"	
	
	"github.com/fuchsi/torrentfile"
)

func main() {
	file, err := os.Open(os.Args[1]) 
    	if err != nil {
    		log.Fatal(err)
    	}
    	defer file.Close()
    	
    	tfile, err := torrentfile.DecodeTorrentFile(file)
    	if err != nil {
    		log.Fatal(err)
    	}
    	
    	fmt.Println("Name: " + tfile.Name)
}
``` 