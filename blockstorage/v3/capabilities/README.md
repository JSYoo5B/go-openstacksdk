# Cinder backend capabilities

`service.Capabilities.GetRecord(ctx, host)` fetches the backend capabilities
without discovery or a prior lookup. The response is the whole JSON body;
there is no capabilities envelope. Resource holds declared fields and
properties/list descriptor values; Wire retains unknown server fields and
Envelope/Header/StatusCode retain actual HTTP evidence. Missing backend errors
remain native errors with the actual response.

[Python and Go usage](python-records.md)
