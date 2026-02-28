package config //nolint:testpackage

import "testing"

func TestMarshalTopics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		topicPrefix string
		want        Topics
	}{
		{
			name:        "simple-prefix",
			topicPrefix: "signal",
			want: Topics{
				Message:   "signal/" + TopicMessageSuffix,
				Status:    "signal/" + TopicOnlineSuffix,
				Connected: "signal/" + TopicConnectedSuffix,
			},
		},
		{
			name:        "keeps-internal-slashes",
			topicPrefix: "signal/api",
			want: Topics{
				Message:   "signal/api/" + TopicMessageSuffix,
				Status:    "signal/api/" + TopicOnlineSuffix,
				Connected: "signal/api/" + TopicConnectedSuffix,
			},
		},
		{
			name:        "trims-spaces-slashes-and-hashes",
			topicPrefix: " #/signal-api/ ",
			want: Topics{
				Message:   "signal-api/" + TopicMessageSuffix,
				Status:    "signal-api/" + TopicOnlineSuffix,
				Connected: "signal-api/" + TopicConnectedSuffix,
			},
		},
		{
			name:        "trims-spaces-slashes-and-hashes-valid",
			topicPrefix: "/signal-api",
			want: Topics{
				Message:   "signal-api/" + TopicMessageSuffix,
				Status:    "signal-api/" + TopicOnlineSuffix,
				Connected: "signal-api/" + TopicConnectedSuffix,
			},
		},
		{
			name:        "trims-spaces-slashes-and-hashes-valid",
			topicPrefix: " #/signal-api/#/ ",
			want: Topics{
				Message:   "signal-api/" + TopicMessageSuffix,
				Status:    "signal-api/" + TopicOnlineSuffix,
				Connected: "signal-api/" + TopicConnectedSuffix,
			},
		},
		{
			name:        "trims-spaces-slashes-and-hashes-2",
			topicPrefix: "#/",
			want: Topics{
				Message:   ClientPrefix + "/message",
				Status:    ClientPrefix + "/online",
				Connected: ClientPrefix + "/connected",
			},
		},
		{
			name:        "trims-spaces-single-slash",
			topicPrefix: "/",
			want: Topics{
				Message:   ClientPrefix + "/message",
				Status:    ClientPrefix + "/online",
				Connected: ClientPrefix + "/connected",
			},
		},
		{
			name:        "trims-spaces-single-space",
			topicPrefix: " ",
			want: Topics{
				Message:   ClientPrefix + "/message",
				Status:    ClientPrefix + "/online",
				Connected: ClientPrefix + "/connected",
			},
		},
		{
			name:        "trims-spaces-single-empty",
			topicPrefix: "",
			want: Topics{
				Message:   ClientPrefix + "/message",
				Status:    ClientPrefix + "/online",
				Connected: ClientPrefix + "/connected",
			},
		},
		{
			name:        "empty-prefix-after-trim",
			topicPrefix: "  ////  ",
			want: Topics{
				Message:   ClientPrefix + "/" + TopicMessageSuffix,
				Status:    ClientPrefix + "/" + TopicOnlineSuffix,
				Connected: ClientPrefix + "/" + TopicConnectedSuffix,
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := marshalTopics(tc.topicPrefix)
			if got == nil {
				t.Fatalf("expected topics, got nil")
			}

			if *got != tc.want {
				t.Fatalf("unexpected topics: got %#v, want %#v", *got, tc.want)
			}
		})
	}
}
