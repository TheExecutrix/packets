# packets

Packets is a go library for streaming packets. The packet starts
with a uvarint specifying the length of the packet.

## CreatePacket and SplitPacket

Create a packet using CreatePacket(), prepending a byte array with
its uvarint length. Read back the packet using SplitPacket.

## WritePacket

WritePacket writes a packet to io.Writer, same as calling
writer.Write(CreatePacket(...)). It's more efficient.

## PacketReader

The PacketReader reads a series of packets from io.Reader. Call Read()
to read each packet, use SetMaxPacketLength to limit the size of a
packet.

## PacketStream

The PacketStream reads packets from io.Reader and writes to a golang
channel. The channel is closed after an error, and the error is in
Err().
