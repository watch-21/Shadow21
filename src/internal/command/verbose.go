package command

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"sync"
)

var verboseOutput struct {
	sync.RWMutex
	writer io.Writer
}

func SetVerboseOutput(writer io.Writer) {
	verboseOutput.Lock()
	defer verboseOutput.Unlock()
	verboseOutput.writer = writer
}

func writeVerboseInvocation(command string, args []string, stdout, stderr []byte) {
	verboseOutput.RLock()
	defer verboseOutput.RUnlock()
	if verboseOutput.writer == nil {
		return
	}

	var output bytes.Buffer
	fmt.Fprintf(&output, "[CMD ] %s", command)
	for _, arg := range args {
		fmt.Fprintf(&output, " %s", strconv.Quote(arg))
	}
	fmt.Fprintln(&output)
	writeVerboseBlock(&output, "stdout", stdout)
	writeVerboseBlock(&output, "stderr", stderr)
	_, _ = verboseOutput.writer.Write(output.Bytes())
}

func writeVerboseBlock(writer io.Writer, name string, data []byte) {
	if len(data) == 0 {
		return
	}
	fmt.Fprintf(writer, "%s:\n", name)
	_, _ = writer.Write(data)
	if data[len(data)-1] != '\n' {
		fmt.Fprintln(writer)
	}
}
