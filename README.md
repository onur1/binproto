# binproto

generic support for binary-based request response protocols.

## Message format

binproto header is a varint encoded unsigned 64-bit integer which consist of an ID (first 60-bits) and a Channel number (last 4-bits). The rest of the message is data.

## Buffered

binproto uses an internal buffer which allocates 4096 bytes by default. You can adjust this value if your protocol requires larger (or smaller) chunks.
