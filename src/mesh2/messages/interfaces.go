package messages

import (
	"github.com/skycoin/skycoin/src/cipher"
)

type NodeInterface interface {
	GetId() cipher.PubKey
	InjectTransportMessage(transportId TransportId, msg []byte)
	SetTransport(TransportId, TransportInterface)
}

type TransportInterface interface {
	InjectNodeMessage([]byte)
}

//later add "transport status" struct
