package logging

import "log/slog"

// Domain identifiers

func Conversation(id string) slog.Attr {
	return slog.String("conversation_id", id)
}

func Sender(id string) slog.Attr {
	return slog.String("sender_id", id)
}

func ClientMsg(id string) slog.Attr {
	return slog.String("client_msg_id", id)
}

func Sequence(seq int64) slog.Attr {
	return slog.Int64("sequence", seq)
}

// Request / tracing

func RequestID(id string) slog.Attr {
	return slog.String("request_id", id)
}

func TraceID(id string) slog.Attr {
	return slog.String("trace_id", id)
}

func SpanID(id string) slog.Attr {
	return slog.String("span_id", id)
}

// Error handling

func Err(err error) slog.Attr {
	if err == nil {
		return slog.String("error", "")
	}
	return slog.String("error", err.Error())
}
