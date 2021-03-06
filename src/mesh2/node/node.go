package node

import (
	"github.com/skycoin/skycoin/src/mesh2/messages"
	"github.com/skycoin/skycoin/src/mesh2/transport"

	"errors"
	"fmt"
	"github.com/skycoin/skycoin/src/cipher"
	"log"
)

//A Node has a map of route rewriting rules
//A Node has a control channel for setting and modifying the route rewrite rules
//A Node has a list of transports

//Route rewriting rules
//-nodes receive messages on a route
//-nodes look up the route in a table and if it has a rewrite rule, rewrites the route
// and forwards it to the transport

type Node struct {
	Id                     cipher.PubKey
	IncomingChannel        chan []byte
	IncomingControlChannel chan messages.InControlMessage

	Transports           map[messages.TransportId]*transport.Transport
	RouteForwardingRules map[messages.RouteId]*RouteRule

	controlChannels map[messages.ChannelId]*ControlChannel
}

type RouteRule struct {
	IncomingTransport messages.TransportId
	OutgoingTransport messages.TransportId
	IncomingRoute     messages.RouteId
	OutgoingRoute     messages.RouteId
}

func NewNode() *Node {
	node := new(Node)
	node.Id = CreatePubKey()
	node.IncomingChannel = make(chan []byte, 1024)
	node.IncomingControlChannel = make(chan messages.InControlMessage, 1024)
	node.Transports = make(map[messages.TransportId]*transport.Transport)
	node.RouteForwardingRules = make(map[messages.RouteId]*RouteRule)
	node.controlChannels = make(map[messages.ChannelId]*ControlChannel)
	fmt.Printf("Created Node %d\n", node.Id)
	return node
}

func CreatePubKey() cipher.PubKey {
	pub, _ := cipher.GenerateKeyPair()
	return pub
}

func (self *Node) Shutdown() {
	close(self.IncomingChannel)
}

//move node forward on tick, process events
func (self *Node) Tick() {
	//process incoming messages
	go self.HandleIncomingTransportMessages() //pop them off the channel
	go self.HandleIncomingControlMessages()   //pop them off the channel
}

func (self *Node) HandleIncomingTransportMessages() {
	for msg := range self.IncomingChannel {
		fmt.Printf("\nnode with id %d accepting a message with type %d\n\n", self.Id, messages.GetMessageType(msg))
		//process our incoming messages
		switch messages.GetMessageType(msg) {
		//InRouteMessage is the only message coming in to node from transports
		case messages.MsgInRouteMessage:
			var m1 messages.InRouteMessage
			messages.Deserialize(msg, &m1)
			fmt.Println("InRouteMessage", m1)
			self.HandleInRouteMessage(m1)
		default:
			fmt.Println("wrong type", messages.GetMessageType(msg))
		}
	}
}

func (self *Node) HandleIncomingControlMessages() {
	for msg := range self.IncomingControlChannel {
		self.HandleControlMessage(msg.ChannelId, msg.PayloadMessage)
	}
}

func (self *Node) HandleInRouteMessage(m1 messages.InRouteMessage) {
	//look in route table
	routeId := m1.RouteId
	transportId := m1.TransportId //who is is from
	//check that transport exists
	if _, ok := self.Transports[transportId]; !ok {
		log.Printf("Node: Received message From Transport that does not exist\n")
	}
	//check if route exists
	routeRule, ok := self.RouteForwardingRules[routeId]
	if !ok {
		log.Printf("Node: Received route message for route that does not exist\n")
	}
	//check that incoming transport is correct
	if transportId != routeRule.IncomingTransport {
		log.Panic("Node: incoming route does not match the transport id it should be received from")
	}
	if routeId != routeRule.IncomingRoute {
		log.Panic("Node: impossible, incoming route id does not match route rule id")
	}
	//construct new packet
	var out messages.OutRouteMessage
	out.RouteId = routeRule.OutgoingRoute              //replace inRoute, with outRoute, using rule
	outgoingTransportId := routeRule.OutgoingTransport //find transport to resend datagram
	out.Datagram = m1.Datagram
	//serialize message, with prefix
	b1 := messages.Serialize(messages.MsgOutRouteMessage, out)
	//self.Transports[transportId].InjectNodeMessage(b1) //inject message to transport
	if outgoingTransport, found := self.Transports[outgoingTransportId]; found {
		outgoingTransport.IncomingChannel <- b1 //inject message to transport
	}
}

//inject an incoming message from the transport
func (self *Node) InjectTransportMessage(transportID messages.TransportId, msg []byte) {
	self.IncomingChannel <- msg //push message to channel
}

func (self *Node) InjectControlMessage(msg messages.InControlMessage) {
	self.IncomingControlChannel <- msg //push message to channel
}

func (self *Node) GetId() cipher.PubKey {
	return self.Id
}

func (self *Node) GetTransportById(id messages.TransportId) (*transport.Transport, error) {
	tr, ok := self.Transports[id]
	if !ok {
		return nil, errors.New(fmt.Sprintf("Node %d doesn't contain transport with id %d", self.Id, id))
	}
	return tr, nil
}

func (self *Node) GetTransportToNode(nodeId cipher.PubKey) (*transport.Transport, error) {
	for _, transport := range self.Transports {
		if nodeId == transport.StubPair.AttachedNode.GetId() {
			return transport, nil
		}
	}
	return nil, errors.New("No transport to node")
}

func (self *Node) SetTransport(id messages.TransportId, tr messages.TransportInterface) {
	self.Transports[id] = tr.(*transport.Transport)
}
