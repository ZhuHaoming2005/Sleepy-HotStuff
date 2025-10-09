package message

import (
	"github.com/vmihailenco/msgpack/v5"
	pb "sleepy-hotstuff/src/proto/communication"
)

type ClientRequest struct {
	Type pb.MessageType
	ID   int64
	OP   []byte // Message payload. Opt for contract.
	TS   int64  // Timestamp
}

func (r *ClientRequest) Serialize() ([]byte, error) {
	jsons, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return jsons, nil
}

func DeserializeClientRequest(input []byte) ClientRequest {
	var clientRequest = new(ClientRequest)
	msgpack.Unmarshal(input, &clientRequest)
	return *clientRequest
}
