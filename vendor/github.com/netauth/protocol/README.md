# NetAuth Protocol

This repo defines the NetAuth wire protocol which is implemented as an
RPC interface on top of gRPC.  This interface allows for remote
functions of the NetAuth server to be accessed by consumers, such as
authenticating an entity, searching for groups, or modifying KV2 data.

Direct consumption of this repo is discouraged, and you are advised to
use one of the published client implementations for NetAuth.  If you
need an implementation that is not in a language already present,
please open a discussion thread in the main NetAuth repo.
