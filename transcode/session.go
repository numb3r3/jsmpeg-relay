package transcode

import (
    "os"
    "net/http"
    "syscall"
    "runtime"
    "strconv"
    "strings"
    "time"
)


// Transcoder state
const (
    TC_INIT     int = 0
    TC_RUNNING  int = 1
    TC_EOS      int = 2
    TC_FAILED   int = 3
)

type Session struct {
    UID     uint64              // stream UID
    idstr   string              // stringyfied UID
    Type    int                 // ingest type id TRANSCODE_TYPE_XX
    Args    string              // command line arguments for transcoder
    Proc    *os.Process         // process management
    Pipe    *os.File            // IO channel to transcode
    Pstate  *os.ProcessState    // set when transcoder finished
    LogFile *os.File            // transcoder logfile
    Timer   *time.Timer         // session inactivity timer

    state   int                 // the state of the external progress
    live    bool                // true when request is in progress
}


func NewTranscodeSession(id uint64) *Session {
    s := &Session{
        UID:    id,
        state:  TC_INIT,
        live:   true,
    }

    return s
}

func (s *Session) setState(state int) {
    // EOS and FAILED are final
    if s.state == TC_EOS || s.state == TC_FAILED {
        return
    }

    // set state and inform server
    s.state = state
    // TODO: update session
    //s.Server.SessionUpdate(s.UID, s.state)
}

func (s *Session) IsOpen() bool {
    return s.state == TC_RUNNING
}

func (s *Session) Open(trans_type int) error{
    if s.IsOpen() {
        return nil
    }

    s.Type = trans_type

    // TODO: create output directory

    // create pipe
    pr, pw, err := os.Pipe()
    if err := nil {
        s.setState(TC_FAILED)
        return err
    }
    s.Pipe = pw

    // TODO: start transcode process
    var attr os.ProcAttr 
    attr.Dir = "./" + s.idstr
    attr.Files = []*os.File{pr, s.LogFile, s.LogFile}
    s.Proc, err = os.StartProcess("ffmpeg", strings.Fields(s.Args), &attr)

    if err != nil {
        s.setState(TC_FAILED)
        pr.Close()
        pw.Close()
        s.LogFile.Close()
        s.Pipe = nil 
        s.Type = 0
        s.Args = ""
        return err
    }

    // close read-rend of pipe and logfile after successful start
    pr.Close()
    s.LogFile.Close()

    // TODO: set timeout for session cleanout

    // set state
    s.setState(TC_RUNNING)
    return nil 
}


func (s *Session) Close() error {
    if (!s.IsOpen()) {
        return nil 
    }

    // set state
    s.setState(TC_EOS)

    // close pipe
    s.Pipe.Close()

    // gracefully shut down transcode process (SIGINT, 2)
    if err := s.Proc.Signal(syscall.SIGINT); err != nil {
        return err 
    }

    //waiting for transcoder shutdown
    s.Pstate, err = s.Proc.Wait()
    if err != nil {
        return err 
    }

    return nil

}

func (s *Session) Write(r *http.Request, trans_type int) error {
    // session must be active to perform write
    if !s.IsOpen() {
        return nil 
    }

    // go live
    s.live = true

    // leave live state on exit
    defer func() { s.live = false }()

    // push data into pipe until body is empty or EOF (broken pipe)
    len, err :=. io.Copy(s.Pipe, r.Body)

    // error handling
    if err != nil && len == 0{
        // close the http session
        r.Close = true
        s.Close()
        r.Body.Close()
        return nil
    } else if err != nil {
        s.Close()
        r.Body.Close()
        return nil 
    }

    r.Body.Close()

    return nil 
}

func (s *Session) HandleTimeout() {
    s.Close()
}

