package protocol

import (
	"bytes"
	"fmt"
	"io"

	"mycelia/comm"
	"mycelia/errgo"
	"mycelia/globals"
)

// -----------------------------------------------------------------------------
// Version 1 object decoding.
// -----------------------------------------------------------------------------
// *Note that this is a messaging protocol, not a file transfer protocol
// -----------------------------------------------------------------------------
// The version 1 protocol looks as follows:

// # Fixed field sized header
// +---------+--------+-------------+-------------+---------------+
// | u32 len | u8 ver | u8 obj_type | u8 cmd_type | u8 ack policy |
// +---------+--------+-------------+-------------+---------------+

// which is then followed by a variable field sized sub-header that contains a
// UID and the sender's address for tracking purposes.

// # Tracking Sub-header
// +-------------+---------------------+
// | u8 len uid  | u16 len return addr |
// +-------------+---------------------+

// which is then followed by 4 uint8 sized byte fields that act as arguments for
// the object type in the fixed header.
// Because these are byte streams, all arguments are considered string types
// unless the executor casts them to another type.

// # Argument Sub-Header
// +---------------+---------------+---------------+---------------+
// |  u8 len arg1  |  u8 len arg2  |  u8 len arg3  |  u8 len arg4  |
// +---------------+---------------+---------------+---------------+

// And finally the message payload that would be delivered to external sources.
// If this is unused because the message is changing the internals of the broker
// at runtime, then the field defaults to a vlaue of 0x00.

// # Globals Body
// +-----------------+
// | u16 len payload |
// +-----------------+

// -----------------------------------------------------------------------------
// Responses are a three field message: message length prefix, corresponding
// uid, and the ack/nack value.

// +---------+------------+--------------+
// | u16 len | u8 len uid | u8 ack value |
// +---------+------------+--------------+

// -----------------------------------------------------------------------------

//--------Decoding--------------------------------------------------------------

func decodeV1(data []byte, resp *comm.ConnResponder) (*Object, error) {
	r := bytes.NewReader(data)
	obj := &Object{}
	obj.Responder = resp

	// ObjType + CmdType
	obj, err := parseBaseHeader(r, obj)
	if err != nil {
		return nil, err
	}
	// UID + Source Address
	obj, err = parseTrackingHeader(r, obj)
	if err != nil {
		return nil, err
	}
	// Arg fields
	obj, err = parseArgumentFields(r, obj)
	if err != nil {
		return nil, err
	}
	// Payload
	payload, err := readBytesU16(r)
	if err != nil {
		wMsg := fmt.Sprintf(
			"Unable to parse payload from %s: %s",
			obj.Responder.C.RemoteAddr().String(), err,
		)
		wErr := errgo.NewError(wMsg, globals.VERB_WRN)
		return nil, wErr
	}
	obj.Payload = payload

	response := &Response{
		UID:     obj.UID,
		AckType: globals.ACK_TYPE_UNKNOWN,
	}
	obj.Response = response

	if r.Len() != 0 {
		obj = nil
		err = errgo.NewError("Unaccounted data in reader", globals.VERB_WRN)
	}

	return obj, err
}

// Parses the header after version: obj_type, and cmd_type from message.
func parseBaseHeader(r io.Reader, cmd *Object) (*Object, error) {
	if err := readU8(r, &cmd.ObjType); err != nil {
		wMsg := fmt.Sprintf(
			"Unable to parse u8 ObjType field from message: %s", err,
		)
		wErr := errgo.NewError(wMsg, globals.VERB_WRN)
		return nil, wErr
	}

	if err := readU8(r, &cmd.CmdType); err != nil {
		wMsg := fmt.Sprintf(
			"Unable to parse u8 CmdType field from message: %s", err,
		)
		wErr := errgo.NewError(wMsg, globals.VERB_WRN)
		return nil, wErr
	}

	if err := readU8(r, &cmd.AckPlcy); err != nil {
		wMsg := fmt.Sprintf(
			"Unable to parse u8 AckPolicy field from message: %s", err,
		)
		wErr := errgo.NewError(wMsg, globals.VERB_WRN)
		return nil, wErr
	}

	return cmd, nil
}

// Parses the UID and sender address from the reader.
func parseTrackingHeader(r io.Reader, cmd *Object) (*Object, error) {
	// UID field comes before sender address field.
	uid, err := readStringU8(r)
	if err != nil {
		wMsg := fmt.Sprintf(
			"Unable to parse string UID field from message: %s", err,
		)
		wErr := errgo.NewError(wMsg, globals.VERB_WRN)
		return nil, wErr
	}
	cmd.UID = uid

	return cmd, nil
}

// Parse the four argument fields from the reader.
func parseArgumentFields(r io.Reader, cmd *Object) (*Object, error) {
	arg1, err := readStringU8(r)
	if err != nil {
		wMsg := fmt.Sprintf("Unable to parse argument position %d for %s: %s",
			1, cmd.Responder.C.RemoteAddr().String(), err,
		)
		wErr := errgo.NewError(wMsg, globals.VERB_WRN)
		return nil, wErr
	}
	cmd.Arg1 = arg1

	arg2, err := readStringU8(r)
	if err != nil {
		wMsg := fmt.Sprintf("Unable to parse argument position %d for %s, %s",
			2, cmd.Responder.C.RemoteAddr().String(), err,
		)
		wErr := errgo.NewError(wMsg, globals.VERB_WRN)
		return nil, wErr
	}
	cmd.Arg2 = arg2

	arg3, err := readStringU8(r)
	if err != nil {
		wMsg := fmt.Sprintf("Unable to parse argument position %d for %s: %s",
			3, cmd.Responder.C.RemoteAddr().String(), err,
		)
		wErr := errgo.NewError(wMsg, globals.VERB_WRN)
		return nil, wErr
	}
	cmd.Arg3 = arg3

	arg4, err := readStringU8(r)
	if err != nil {
		wMsg := fmt.Sprintf("Unable to parse argument position %d for %s: %s",
			4, cmd.Responder.C.RemoteAddr().String(), err,
		)
		wErr := errgo.NewError(wMsg, globals.VERB_WRN)
		return nil, wErr
	}
	cmd.Arg4 = arg4

	return cmd, nil
}

//--------Encoding--------------------------------------------------------------

// Encodes a protocol.Response object into []byte.
func EncodeResponseV1(response Response) []byte {
	body := bytes.NewBuffer(nil)
	_ = writeString8(body, response.UID)
	writeU8(body, response.AckType)

	full := bytes.NewBuffer(nil)
	writeU16(full, uint16(body.Len()))
	full.Write(body.Bytes())
	return full.Bytes()
}
