package traefik

import "testing"

func Test_getPath(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "has path prefix",
			args:    args{s: "PathPrefix(`/hello-world`)"},
			want:    "/hello-world",
			wantErr: false,
		},
		{
			name:    "has path",
			args:    args{s: "Path(`/hello-world`)"},
			want:    "/hello-world",
			wantErr: false,
		},
		{
			name:    "has path",
			args:    args{s: "Path(wrong)"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getPath(tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("getPath() error = %v, wantErr %v", err, tt.wantErr)

				return
			}
			if got != tt.want {
				t.Errorf("getPath() got = %v, want %v", got, tt.want)
			}
		})
	}
}
