### Setting up the tracer
> #### All in one Docker image
> This image, designed for quick local testing, launches the Jaeger UI, collector, query, and agent, with an in memory storage component.
> You can then navigate to http://localhost:16686 to access the Jaeger UI.

```
$ docker run -d --name jaeger -p 6831:6831/udp -p 6832:6832/udp -p 5778:5778 -p 16686:16686 -p 14268:14268 jaegertracing/all-in-one:latest
```

**NOTE**: For trace `RPCServer` to show up in the UI, the rpc server needs to terminate first in order for the span to finish.

### snapshots of the tracer UI

#### Search by service

service name, e.g., `RPCServer@x.x.x.x:y`

![](./images/search_by_service.png)

#### Search by operation

operation name e.g. `RPCServer`, `RPCServer.AddPeer`, `RPCServer.BroadcastCollation`, etc.

![](./images/search_by_operation.png)

#### Overview of one trace

![](./images/overview_of_one_trace.png)

#### Overview of one span

Click on certain span to expand and get an overview of the span

![](./images/overview_of_one_span.png)

#### Detail of one span

Click on the info, e.g., `Tags`, `Process`, `Logs`, to expand and get more detail regarding the span

![](./images/detail_of_one_span.png)
