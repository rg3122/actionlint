package actionlint

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseContainerPortsAndVolumes(t *testing.T) {
	input := []byte(`on: push
jobs:
  test:
    runs-on: ubuntu-latest
    container:
      image: alpine:3
      ports:
        - "8080:80"
      volumes:
        - "/tmp:/tmp"
    services:
      redis:
        image: redis:7
        ports:
          - "6379:6379"
        volumes:
          - "/data:/data"
    steps:
      - run: echo hello
`)

	wf, errs := Parse(input)
	if len(errs) != 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	job := wf.Jobs["test"]
	if job == nil {
		t.Fatal("job was not parsed")
	}

	assertContainer := func(name string, c *Container, wantPort, wantVolume string) {
		t.Helper()
		if c == nil {
			t.Fatalf("%s container was not parsed", name)
		}
		if len(c.Ports) != 1 || c.Ports[0].Value != wantPort {
			t.Errorf("%s ports = %v, want [%q]", name, c.Ports, wantPort)
		}
		if len(c.Volumes) != 1 || c.Volumes[0].Value != wantVolume {
			t.Errorf("%s volumes = %v, want [%q]", name, c.Volumes, wantVolume)
		}
	}

	assertContainer("job", job.Container, "8080:80", "/tmp:/tmp")
	if job.Services == nil {
		t.Fatal("services were not parsed")
	}
	redis := job.Services.Value["redis"]
	if redis == nil {
		t.Fatal("redis service was not parsed")
	}
	assertContainer("service", redis.Container, "6379:6379", "/data:/data")
}

func BenchmarkParseWorkflow(b *testing.B) {
	type bench struct {
		name  string
		input []byte
	}

	loadBench := func(name string) bench {
		i, err := os.ReadFile(filepath.Join("testdata", "bench", name+".yaml"))
		if err != nil {
			b.Fatal(err)
		}
		return bench{name, i}
	}

	for _, bc := range []bench{
		loadBench("minimal"),
		loadBench("small"),
		loadBench("large"),
	} {
		b.Run(bc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if _, errs := Parse(bc.input); len(errs) > 0 {
					b.Fatal(errs)
				}
			}
		})
	}
}

func BenchmarkParseTestData(b *testing.B) {
	type bench struct {
		name   string
		inputs [][]byte
	}

	loadBench := func(name string) bench {
		inputs := [][]byte{}
		_, fs, err := testFindAllWorkflowsInDir(name)
		if err != nil {
			b.Fatal(err)
		}
		for _, f := range fs {
			bs, err := os.ReadFile(f)
			if err != nil {
				b.Fatal(err)
			}
			inputs = append(inputs, bs)
		}
		return bench{name, inputs}
	}

	for _, bc := range []bench{
		loadBench("examples"),
		loadBench("ok"),
		loadBench("err"),
	} {
		b.Run(bc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				for _, in := range bc.inputs {
					// Note: Some workflows may cause parse error
					Parse(in)
				}
			}
		})
	}
}
