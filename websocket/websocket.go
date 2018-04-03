package websocket

import (
    "io"
    "net"
    "net/http"
    "sync"
    "time"

    "github.com/gorilla/websocket"
    "github.com/numb3r3/jsmpeg-relay/log"
)

type websocketConn interface {
    NextReader() (messageType int, r io.Reader, err error)
    NextWriter(messageType int) (io.WriteCloser, error)
    Close() error
    LocalAddr() net.Addr
    RemoteAddr() net.Addr
    SetReadDeadline(t time.Time) error
    SetWriteDeadline(t time.Time) error
    SetCloseHandler(h func(code int, text string) error)
}

// websocketConn represents a websocket connection.
type websocketTransport struct {
    sync.Mutex
    socket  websocketConn
    reader  io.Reader
    closing chan bool
}

const (
    writeWait        = 10 * time.Second    // Time allowed to write a message to the peer.
    pongWait         = 60 * time.Second    // Time allowed to read the next pong message from the peer.
    pingPeriod       = (pongWait * 9) / 10 // Send pings to peer with this period. Must be less than pongWait.
    closeGracePeriod = 10 * time.Second    // Time to wait before force close on connection.
)

// The default upgrader to use
var upgrader = &websocket.Upgrader{
    CheckOrigin:  func(r *http.Request) bool { return true },
}

// TryUpgrade attempts to upgrade an HTTP request to mqtt over websocket.
func TryUpgrade(w http.ResponseWriter, r *http.Request) (*websocketTransport, bool) {
    if w == nil || r == nil {
        return nil, false
    }

    if ws, err := upgrader.Upgrade(w, r, nil); err == nil {
        return newWebsocketConn(ws), true
    }

    return nil, false
}

// newWebsocketConn creates a new transport from websocket.
func newWebsocketConn(ws websocketConn) *websocketTransport {
    conn := &websocketTransport{
        socket:  ws,
        closing: make(chan bool),
    }

    ws.SetCloseHandler(func(code int, text string) (err error) {
        conn.closing <- true
        if err := conn.Close(); err != nil {
            logging.Error("callback: websocket could not be closed: ", err)
        }
        close(conn.closing)
        logging.Debug("callback: websocket connection closed")
        return
    })

    /*ws.SetReadLimit(maxMessageSize)
    ws.SetReadDeadline(time.Now().Add(pongWait))
    ws.SetPongHandler(func(string) error { ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })
    ws.SetCloseHandler(func(code int, text string) error {
        return conn.Close()
    })
    utils.Repeat(func() {
        log.Println("ping")
        if err := ws.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(writeWait)); err != nil {
            log.Println("ping:", err)
        }
    }, pingPeriod, conn.closing)*/

    return conn
}

// Read reads data from the connection. It is possible to allow reader to time
// out and return a Error with Timeout() == true after a fixed time limit by
// using SetDeadline and SetReadDeadline on the websocket.
func (c *websocketTransport) Read(b []byte) (n int, err error) {
    var opCode int
    if c.reader == nil {
        // New message
        var r io.Reader
        for {
            if opCode, r, err = c.socket.NextReader(); err != nil {
                return
            }

            if opCode != websocket.BinaryMessage && opCode != websocket.TextMessage {
                continue
            }

            c.reader = r
            break
        }
    }

    // Read from the reader
    n, err = c.reader.Read(b)
    if err != nil {
        if err == io.EOF {
            c.reader = nil
            err = nil
        }
    }
    return
}

// Write writes data to the connection. It is possible to allow writer to time
// out and return a Error with Timeout() == true after a fixed time limit by
// using SetDeadline and SetWriteDeadline on the websocket.
func (c *websocketTransport) Write(b []byte) (n int, err error) {
    // Serialize write to avoid concurrent write
    c.Lock()
    defer c.Unlock()

    var w io.WriteCloser
    if w, err = c.socket.NextWriter(websocket.BinaryMessage); err == nil {
        if n, err = w.Write(b); err == nil {
            err = w.Close()
        }
    }
    return
}

func(c *websocketTransport) Closing() <-chan bool{
    return c.closing
}

// Close terminates the connection.
func (c *websocketTransport) Close() (err error) {
    c.closing <- true
    if err := c.socket.Close(); err != nil {
        logging.Error("websocket could not be closed: ", err)
    }
    close(c.closing)
    logging.Debug("websocket connection closed")
    // return c.socket.Close()
    return
}

// LocalAddr returns the local network address.
func (c *websocketTransport) LocalAddr() net.Addr {
    return c.socket.LocalAddr()
}

// RemoteAddr returns the remote network address.
func (c *websocketTransport) RemoteAddr() net.Addr {
    return c.socket.RemoteAddr()
}

// SetDeadline sets the read and write deadlines associated
// with the connection. It is equivalent to calling both
// SetReadDeadline and SetWriteDeadline.
func (c *websocketTransport) SetDeadline(t time.Time) (err error) {
    if err = c.socket.SetReadDeadline(t); err == nil {
        err = c.socket.SetWriteDeadline(t)
    }
    return
}

// SetReadDeadline sets the deadline for future Read calls
// and any currently-blocked Read call.
func (c *websocketTransport) SetReadDeadline(t time.Time) error {
    return c.socket.SetReadDeadline(t)
}

// SetWriteDeadline sets the deadline for future Write calls
// and any currently-blocked Write call.
func (c *websocketTransport) SetWriteDeadline(t time.Time) error {
    return c.socket.SetWriteDeadline(t)
}
