package message

import (
	"github.com/vmihailenco/msgpack/v5"
	"sleepy-hotstuff/src/cryptolib"
	pb "sleepy-hotstuff/src/proto/communication"
)

type ReplicaMessage struct {
	Mtype    TypeOfMessage
	Instance int
	Source   int64
	Hash     []byte
	TS       int64
	Payload  []byte //message payload
	Value    int
	Maj      int
	Round    int
	Epoch    int
}

type HotStuffMessage struct {
	Mtype     pb.MessageType
	Seq       int
	Source    int64
	View      int
	Conf      int
	OPS       []pb.RawMessage
	MemOPS    []pb.RawMessage
	Hash      []byte
	PreHash   []byte
	Sig       []byte
	QC        []byte
	LQC       []byte
	ComBlocks []byte
	Hashes    map[string]int
	SidInfo   []int64
	TS        int64
	Num       int
	Epoch     int
	Count     int
	V         []MessageWithSignature
}

// MembershipInfo Used for dynamic membership only
type MembershipInfo struct {
	IP   string
	Port string
	TS   int64
	Key  []byte
}

type JoinMessage struct {
	Mtype   pb.MessageType
	View    int
	Source  int64
	Epoch   int
	Hash    []byte
	MInfo   MembershipInfo
	Address string
}

// MemMessage Membership message. Used for dynamic membership only
type MemMessage struct {
	ID   int
	JMsg JoinMessage
}

/*
Serialize MemMessage
*/
func (r *MemMessage) Serialize() ([]byte, error) {
	ser, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return ser, nil
}

/*
Deserialize []byte to MemMessage
*/
func DeserializeMemMessage(input []byte) MemMessage {
	var memMessage = new(MemMessage)
	msgpack.Unmarshal(input, &memMessage)
	return *memMessage
}

/*
Serialize JoinMessage
*/
func (r *JoinMessage) Serialize() ([]byte, error) {
	jsons, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return jsons, nil
}

/*
Deserialize []byte to JoinMessage
*/
func DeserializeJoinMessage(input []byte) JoinMessage {
	var joinMessage = new(JoinMessage)
	msgpack.Unmarshal(input, &joinMessage)
	return *joinMessage
}

/*
Serialize MembershipInfo
*/
func (r *MembershipInfo) Serialize() ([]byte, error) {
	ser, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return ser, nil
}

/*
Get hash of the message
*/
func (r *MembershipInfo) GetIdentifier() []byte {
	jsons, err := r.Serialize()
	if err != nil {
		return []byte("")
	}
	return cryptolib.GenHash(jsons)
}

/*
Get hash of the entire batch
*/
func (r *HotStuffMessage) GetMsgHash() []byte {
	if len(r.OPS) == 0 {
		return []byte("")
	}
	return cryptolib.GenBatchHash(r.OPS)
}

/*
Serialize ReplicaMessage
*/
func (r *HotStuffMessage) Serialize() ([]byte, error) {
	ser, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return ser, nil
}

func DeserializeHotStuffMessage(input []byte) HotStuffMessage {
	var hotStuffMessage = new(HotStuffMessage)
	msgpack.Unmarshal(input, &hotStuffMessage)
	return *hotStuffMessage
}

// ViewChangeMessage VIEWCHANGE and NEWVIEW messages
type ViewChangeMessage struct {
	Mtype    pb.MessageType
	View     int
	Conf     int            //Dynamic membership only. Configuration number.
	ConfAddr map[int]string //Dynamic membership only. Address of the nodes.
	Seq      int
	Source   int64
	P        map[int]Cer                  //Prepare certificate
	PP       map[int]HotStuffMessage      //Pre-prepare message. Used for dynamic membership only.
	E        []int                        //Dynamic membership only. Current membership group
	O        map[int]MessageWithSignature // Operations. Used in new-view message only
	V        []MessageWithSignature       // A quorum of view-change messages. Used in new-view message only
}

type QCBlock struct {
	View       int
	Height     int //height
	Hash       []byte
	PreHash    []byte
	PrePreHash []byte
	QC         [][]byte
	Aux        []byte
	AuxQC      []byte
	IDs        []int64
	TXS        []MessageWithSignature
}

type Transaction struct {
	From  string `json:from`
	To    string `json:to`
	Value int    `json:value`
	// Timestamp int64  `json:timestamp`
}

func (r *Transaction) Serialize() ([]byte, error) {
	txser, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return txser, nil
}

func (r *Transaction) Deserialize(input []byte) error {
	return msgpack.Unmarshal(input, r)
}

func (r *QCBlock) Serialize() ([]byte, error) {
	msgser, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return msgser, nil
}

func (r *QCBlock) Deserialize(input []byte) error {
	return msgpack.Unmarshal(input, r)
}

func DeserializeQCBlock(input []byte) QCBlock {
	var qcblock = new(QCBlock)
	msgpack.Unmarshal(input, qcblock)
	return *qcblock
}

/*
Create MessageWithSignature where the signature of each message (in ReplicaMessage) is attached.
The ReplicaMessage is first serialized into bytes and then signed.
Input

	tmpmsg: ReplicaMessage

Output

	MessageWithSignature: the struct
*/
func CreateMessageWithSig(tmpmsg HotStuffMessage) MessageWithSignature {
	tmpmsgSer, err := tmpmsg.Serialize()
	if err != nil {
		var emptymsg MessageWithSignature
		return emptymsg
	}

	op := MessageWithSignature{
		Msg: tmpmsgSer,
		Sig: cryptolib.GenSig(tmpmsg.Source, tmpmsgSer),
	}
	return op
}

/*
Deserialize []byte to ViewChangeMessage
*/
func DeserializeViewChangeMessage(input []byte) ViewChangeMessage {
	var viewChangeMessage = new(ViewChangeMessage)
	msgpack.Unmarshal(input, &viewChangeMessage)
	return *viewChangeMessage
}

/*
Serialize ViewChangeMessage
*/
func (r *ViewChangeMessage) Serialize() ([]byte, error) {
	jsons, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return jsons, nil
}

// Cer PREPARE certificates and COMMIT certificates
type Cer struct {
	Msgs [][]byte
}

/*
Add a message to the certificate
*/
func (r *Cer) Add(msg []byte) {
	r.Msgs = append(r.Msgs, msg)
}

/*
Get messages in a certificate
*/
func (r *Cer) GetMsgs() [][]byte {
	return r.Msgs
}

/*
Get the number of messages in the certiicate
*/
func (r *Cer) Len() int {
	return len(r.Msgs)
}

type MessageWithSignature struct {
	Msg []byte
	Sig []byte
}

type RawOPS struct {
	OPS []pb.RawMessage
}

type CBCMessage struct {
	Value         map[int][]byte
	RawData       [][]byte
	MerkleBranch  [][][]byte
	MerkleIndexes [][]int64
}

type Signatures struct {
	Hash []byte
	Sigs [][]byte
	IDs  []int64
}

func (r *Signatures) Serialize() ([]byte, error) {
	jsons, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return jsons, nil
}

func DeserializeSignatures(input []byte) ([]byte, [][]byte, []int64) {
	var sigs = new(Signatures)
	msgpack.Unmarshal(input, &sigs)
	return sigs.Hash, sigs.Sigs, sigs.IDs
}

func (r *CBCMessage) Serialize() ([]byte, error) {
	jsons, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return jsons, nil
}

func DeserializeCBCMessage(input []byte) CBCMessage {
	var cbcMessage = new(CBCMessage)
	msgpack.Unmarshal(input, &cbcMessage)
	return *cbcMessage
}

func (r *RawOPS) Serialize() ([]byte, error) {
	jsons, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return jsons, nil
}

/*
Deserialize []byte to MessageWithSignature
*/
func DeserializeRawOPS(input []byte) RawOPS {
	var rawOPS = new(RawOPS)
	msgpack.Unmarshal(input, &rawOPS)
	return *rawOPS
}

/*
Serialize MessageWithSignature
*/
func (r *MessageWithSignature) Serialize() ([]byte, error) {
	jsons, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return jsons, nil
}

/*
Deserialize []byte to MessageWithSignature
*/
func DeserializeMessageWithSignature(input []byte) MessageWithSignature {
	var messageWithSignature = new(MessageWithSignature)
	msgpack.Unmarshal(input, &messageWithSignature)
	return *messageWithSignature
}

func (r *ReplicaMessage) Serialize() ([]byte, error) {
	jsons, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return jsons, nil
}

func DeserializeReplicaMessage(input []byte) ReplicaMessage {
	var replicaMessage = new(ReplicaMessage)
	msgpack.Unmarshal(input, &replicaMessage)
	return *replicaMessage
}

func SerializeWithSignature(id int64, msg []byte) ([]byte, error) {
	request := MessageWithSignature{
		Msg: msg,
		Sig: cryptolib.GenSig(id, msg),
	}

	requestSer, err := request.Serialize()
	if err != nil {
		return []byte(""), err
	}
	return requestSer, err
}

func SerializeWithMAC(id int64, dest int64, msg []byte) ([]byte, error) {
	CBCEncryptor := cryptolib.CBCEncrypterAES(msg)
	request := MessageWithSignature{
		Msg: CBCEncryptor,
		Sig: cryptolib.GenMAC(id, CBCEncryptor),
	}

	requestSer, err := request.Serialize()
	if err != nil {
		return []byte(""), err
	}
	return requestSer, err
}
