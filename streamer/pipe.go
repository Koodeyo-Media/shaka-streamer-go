package streamer

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/google/uuid"
	// "golang.org/x/sys/windows"
)

// A class that represents a pipe.
type Pipe struct {
	readPipeName  string
	writePipeName string
	// readHandle    windows.Handle
	// writeHandle   windows.Handle
	// recvBuffer    chan []byte
}

// Initializes a non-functioning pipe.
func NewPipe() Pipe {
	return Pipe{
		readPipeName:  "",
		writePipeName: "",
		// recvBuffer:    make(chan []byte),
	}
}

/*
A static method used to create a pipe between two processes.

	On POSIX systems, it creates a named pipe using `os.mkfifo`.

	On Windows platforms, it starts a backgroud thread that transfars data from the
	writer to the reader process it is connected to.
*/
func (p *Pipe) CreateIpcPipe(tempDir string, suffix string) {
	uniqueName := uuid.New().String() + suffix
	if runtime.GOOS == "windows" {
		// // Create pipe name.
		// pipeName := "-nt-shaka-" + uniqueName

		// // The read pipe is connected to a writer process.
		// p.readPipeName = `\\.\pipe\W` + pipeName

		// // The write pipe is connected to a reader process.
		// p.writePipeName = `\\.\pipe\R` + pipeName

		// // Set buffer size.
		// bufSize := uint32(64 * 1024)

		// // Create read side of named pipe.
		// readSide, err := windows.CreateNamedPipe(
		// 	p.readPipeName,
		// 	windows.PIPE_ACCESS_INBOUND,
		// 	windows.PIPE_TYPE_BYTE|windows.PIPE_READMODE_BYTE|windows.PIPE_WAIT,
		// 	1,
		// 	bufSize,
		// 	bufSize,
		// 	0,
		// 	nil,
		// )
		// if err != nil {
		// 	panic(fmt.Errorf("failed to create named pipe: %v", err))
		// }

		// p.readHandle = windows.Handle(readSide)

		// // Create write side of named pipe.
		// writeSide, err := windows.CreateNamedPipe(
		// 	p.writePipeName,
		// 	windows.PIPE_ACCESS_OUTBOUND,
		// 	windows.PIPE_TYPE_BYTE|windows.PIPE_READMODE_BYTE|windows.PIPE_WAIT,
		// 	1,
		// 	bufSize,
		// 	bufSize,
		// 	0,
		// 	nil,
		// )
		// if err != nil {
		// 	panic(fmt.Errorf("failed to create named pipe: %v", err))
		// }

		// p.writeHandle = windows.Handle(writeSide)

		// // Start the thread.
		// go p.winThreadFn(bufSize)
	} else {
		pipeName := filepath.Join(tempDir, uniqueName)
		p.readPipeName = pipeName
		p.writePipeName = pipeName
		readableByOwnerOnly := os.FileMode(0600)
		if err := syscall.Mkfifo(pipeName, uint32(readableByOwnerOnly)); err != nil {
			panic(fmt.Errorf("failed to create pipe: %v", err))
		}
	}
}

// Returns a Pipe object whose read or write end is a path to a file.
func (p *Pipe) CreateFilePipe(path string, mode string) {
	// A process will write on the read pipe(file)
	if mode == "w" {
		p.readPipeName = path
		// A process will read from the write pipe(file).
	} else if mode == "r" {
		p.writePipeName = path
	} else {
		panic(fmt.Errorf("'%s' is not a valid mode for a Pipe", mode))
	}
}

// func (p *Pipe) winThreadFn(bufSize uint32) {
// 	// Connect read side of named pipe.
// 	if err := windows.ConnectNamedPipe(p.readHandle, nil); err != nil {
// 		panic(fmt.Errorf("failed to connect named pipe: %v", err))
// 	}

// 	// Connect write side of named pipe.
// 	if err := windows.ConnectNamedPipe(p.writeHandle, nil); err != nil {
// 		panic(fmt.Errorf("failed to connect named pipe: %v", err))
// 	}

// 	// Start reading from pipe.
// 	for {
// 		var buf [64 * 1024]byte
// 		var bytesReturned uint32
// 		if err := windows.ReadFile(p.readHandle, buf[:], &bytesReturned, nil); err != nil {
// 			panic(fmt.Errorf("failed to read from named pipe: %v", err))
// 		}

// 		p.recvBuffer <- buf[:bytesReturned]
// 	}
// }

// Returns a pipe/file path that a reader process can read from.
func (p *Pipe) ReadEnd() string {
	return p.writePipeName
}

// Returns a pipe/file path that a writer process can write to.
func (p *Pipe) WriteEnd() string {
	return p.readPipeName
}
