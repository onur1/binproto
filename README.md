# binproto

binproto implements generic support for binary-based request response protocols.

## Message format

Over the wire messages are length-prefixed and packed in the following format:

```
<varint - length of rest of message>
  <varint - header>
  <payload>
```

Each message starts with an header which is a varint encoded unsigned 64-bit integer which consists of an ID (first 60-bits) and a Channel number (last 4-bits), the rest of the message is payload.


## Buffered

binproto uses an internal buffer which allocates 4096 bytes by default. You can adjust this value if your protocol requires larger (or smaller) chunks.
