package ocn_receiver

import (
	"encoding/json"
	"strings"
	"testing"
)

var body string = `
    {
     "kind": "storage#object",
     "id": "cp300demo1-ocn/activate/1437652340350000",
     "selfLink": "https://www.googleapis.com/storage/v1/b/cp300demo1-ocn/o/activate",
     "name": "activate",
     "bucket": "cp300demo1-ocn",
     "generation": "1437652340350000",
     "metageneration": "1",
     "contentType": "application/octet-stream",
     "updated": "2015-07-23T11:52:20.349Z",
     "storageClass": "STANDARD",
     "size": "1013",
     "md5Hash": "iT13LEmGb6TKsABQMTvFHw==",
     "mediaLink": "https://www.googleapis.com/download/storage/v1/b/cp300demo1-ocn/o/activate?generation=1437652340350000&alt=media",
     "owner": {
      "entity": "user-00b4903a97ed1f23bc6c49b74d47aaf29cabf5b359e1dbb1d98519b04b3667c2",
      "entityId": "00b4903a97ed1f23bc6c49b74d47aaf29cabf5b359e1dbb1d98519b04b3667c2"
     },
     "crc32c": "ePTYgQ==",
     "etag": "CLC4uqiY8cYCEAE="
    }
`

func TestDecode(t *testing.T) {
	r := strings.NewReader(body)
	var m OCNMessage
	err := json.NewDecoder(r).Decode(&m)
	if err != nil {
		t.Error(err)
	}
}
