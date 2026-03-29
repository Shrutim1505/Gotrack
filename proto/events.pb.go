package proto

// EventInput represents a single event to ingest.
type EventInput struct {
	SourceId  string `protobuf:"bytes,1,opt,name=source_id,json=sourceId,proto3" json:"source_id,omitempty"`
	Type      string `protobuf:"bytes,2,opt,name=type,proto3" json:"type,omitempty"`
	Payload   string `protobuf:"bytes,3,opt,name=payload,proto3" json:"payload,omitempty"`
	Timestamp string `protobuf:"bytes,4,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
}

func (x *EventInput) Reset()         { *x = EventInput{} }
func (x *EventInput) String() string { return x.SourceId }
func (x *EventInput) ProtoMessage()  {}

func (x *EventInput) GetSourceId() string  { return x.SourceId }
func (x *EventInput) GetType() string      { return x.Type }
func (x *EventInput) GetPayload() string   { return x.Payload }
func (x *EventInput) GetTimestamp() string  { return x.Timestamp }

// IngestRequest is a batch of events with an API key.
type IngestRequest struct {
	Events []*EventInput `protobuf:"bytes,1,rep,name=events,proto3" json:"events,omitempty"`
	ApiKey string        `protobuf:"bytes,2,opt,name=api_key,json=apiKey,proto3" json:"api_key,omitempty"`
}

func (x *IngestRequest) Reset()         { *x = IngestRequest{} }
func (x *IngestRequest) String() string { return "" }
func (x *IngestRequest) ProtoMessage()  {}

func (x *IngestRequest) GetEvents() []*EventInput { return x.Events }
func (x *IngestRequest) GetApiKey() string         { return x.ApiKey }

// IngestResponse is the result of an ingestion operation.
type IngestResponse struct {
	Accepted    int32    `protobuf:"varint,1,opt,name=accepted,proto3" json:"accepted,omitempty"`
	Rejected    int32    `protobuf:"varint,2,opt,name=rejected,proto3" json:"rejected,omitempty"`
	Duplicates  int32    `protobuf:"varint,3,opt,name=duplicates,proto3" json:"duplicates,omitempty"`
	RejectedIds []string `protobuf:"bytes,4,rep,name=rejected_ids,json=rejectedIds,proto3" json:"rejected_ids,omitempty"`
}

func (x *IngestResponse) Reset()         { *x = IngestResponse{} }
func (x *IngestResponse) String() string { return "" }
func (x *IngestResponse) ProtoMessage()  {}

func (x *IngestResponse) GetAccepted() int32      { return x.Accepted }
func (x *IngestResponse) GetRejected() int32      { return x.Rejected }
func (x *IngestResponse) GetDuplicates() int32    { return x.Duplicates }
func (x *IngestResponse) GetRejectedIds() []string { return x.RejectedIds }
