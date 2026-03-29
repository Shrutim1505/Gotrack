package proto

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
)

// EventServiceServer is the server API for EventService.
type EventServiceServer interface {
	IngestEvents(context.Context, *IngestRequest) (*IngestResponse, error)
	StreamEvents(EventService_StreamEventsServer) error
}

// UnimplementedEventServiceServer provides default implementations.
type UnimplementedEventServiceServer struct{}

func (UnimplementedEventServiceServer) IngestEvents(context.Context, *IngestRequest) (*IngestResponse, error) {
	return nil, fmt.Errorf("IngestEvents not implemented")
}
func (UnimplementedEventServiceServer) StreamEvents(EventService_StreamEventsServer) error {
	return fmt.Errorf("StreamEvents not implemented")
}

// EventService_StreamEventsServer is the server-side streaming interface.
type EventService_StreamEventsServer interface {
	SendAndClose(*IngestResponse) error
	Recv() (*EventInput, error)
	grpc.ServerStream
}

type eventServiceStreamEventsServer struct {
	grpc.ServerStream
}

func (x *eventServiceStreamEventsServer) SendAndClose(m *IngestResponse) error {
	return x.ServerStream.SendMsg(m)
}

func (x *eventServiceStreamEventsServer) Recv() (*EventInput, error) {
	m := new(EventInput)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// RegisterEventServiceServer registers the service with the gRPC server.
func RegisterEventServiceServer(s *grpc.Server, srv EventServiceServer) {
	s.RegisterService(&_EventService_serviceDesc, srv)
}

func _EventService_IngestEvents_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, _ grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(IngestRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	return srv.(EventServiceServer).IngestEvents(ctx, in)
}

func _EventService_StreamEvents_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(EventServiceServer).StreamEvents(&eventServiceStreamEventsServer{stream})
}

var _EventService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "gotrack.EventService",
	HandlerType: (*EventServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "IngestEvents", Handler: _EventService_IngestEvents_Handler},
	},
	Streams: []grpc.StreamDesc{
		{StreamName: "StreamEvents", Handler: _EventService_StreamEvents_Handler, ClientStreams: true},
	},
	Metadata: "events.proto",
}
