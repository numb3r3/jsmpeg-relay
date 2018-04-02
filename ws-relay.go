package main

import (
    "context"
    "flag"
    "net/http"
    "os"
    "os/signal"
    "time"
    "io"

    "github.com/gorilla/mux"
    "github.com/numb3r3/jsmpeg-relay/pubsub"
    "github.com/numb3r3/jsmpeg-relay/log"
    "github.com/numb3r3/jsmpeg-relay/websocket"
)

//Broker default
var broker = pubsub.NewBroker()

// var newclients chan *WsClient
// var bufCh chan []byte

// type WsClient struct {
//  *websocket.Conn
//  data chan []byte
// }

// func NewWsClient(c *websocket.Conn) *WsClient {
//  return &WsClient{c, make(chan []byte, 16)}
// }

// func (c *WsClient) Send(buf []byte) {
//  select {
//  case <-c.data:
//  default:
//  }
//  c.data <- buf
// }

// func (c *WsClient) worker() {
//  for {
//      buf := <-c.data
//      err := c.WriteMessage(websocket.BinaryMessage, buf)
//      if err != nil {
//          c.Close()
//          return
//      }
//  }
// }

// func broacast() {
//  clients := make(map[*WsClient]*WsClient, 0)

//  for {
//      data := <-bufCh
//      for _, c := range clients {
//          c.Send(data)
//      }
//      for len(newclients) > 0 {
//          c := <-newclients
//          clients[c] = c
//          logging.Debug("[ws] [new client]", c.RemoteAddr())
//      }
//  }
// }

// func wsHandler(w http.ResponseWriter, r *http.Request) {
//  c, err := upgrader.Upgrade(w, r, nil)
//  if err != nil {
//      logging.Error("[ws] upgrade failed:", err)
//      return
//  }
//  defer c.Close()

//  client := NewWsClient(c)
//  newclients <- client

//  client.worker()
// }

// func streamHandler(w http.ResponseWriter, r *http.Request) {
//  if r.Body != nil {
//      logging.Info("[stream] [new]", r.RemoteAddr)

//      buf := make([]byte, 1024*1024)
//      for {
//          n, err := r.Body.Read(buf)
//          logging.Info("[stream][recv]", n, err)
//          if err != nil {
//              logging.Info("[stream][recv]err:", err)
//              return
//          }
//          bufCh <- buf[:n]
//      }
//  }
// }

func publishHandler(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    app_name := vars["app_name"]
    stream_key := vars["stream_key"]
    // logging.Infof("publish stream %v / %v", app_name, stream_key)
    if r.Body != nil {
        logging.Debugf("publishing stream %v / %v from %v", app_name, stream_key, r.RemoteAddr)
        
        buf := make([]byte, 1024*1024)
        for {
            n, err := r.Body.Read(buf)
            if err == io.EOF {
                break
            }
            // logging.Info("[stream][recv]", n, err)
            if err != nil {
                logging.Error("[stream][recv] error:", err)
                return
            }
            if n > 0 {
                // logging.Info("broadcast stream")
                broker.Broadcast(buf[:n], app_name + "/" + stream_key)
            }
        }
    }
    r.Body.Close()
    w.WriteHeader(200)
    flusher := w.(http.Flusher)
    flusher.Flush()
    
}

func playHandler(w http.ResponseWriter, r *http.Request){
    vars := mux.Vars(r)
    app_name := vars["app_name"]
    stream_key := vars["stream_key"]
    logging.Infof("play stream %v / %v", app_name, stream_key)

    c, ok := websocket.TryUpgrade(w, r)
    if ok != true {
        logging.Error("[ws] upgrade failed")
        return 
    }
    defer c.Close()

    logging.Info("client remote addr: ", c.RemoteAddr())

    subscriber, err := broker.Attach()
    if err != nil {
        c.Close()
        logging.Error("subscribe error: ", err)
        return
    }
    
    broker.Subscribe(subscriber, app_name + "/" + stream_key)
    for  {
        select {
        case <- c.Closing():
            // logging.Info("subscriber end")
            broker.Detach(subscriber)
        case msg := <-subscriber.GetMessages():
            // logging.Info("[stream][send]")
            data := msg.GetData()
            _, err := c.Write(data)
            if err != nil {
                c.Close()
                return
            }
        }
    }
    return
}



func main() {
    var localAddr = flag.String("l", ":8080", "")
    var wait time.Duration
    flag.DurationVar(&wait, "graceful-timeout", time.Second * 15, "the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
    flag.Parse()

    logging.Info("start ws-relay ....")
    logging.Infof("server listen @ %v", *localAddr)

    r := mux.NewRouter()
    r.HandleFunc("/publish/{app_name}/{stream_key}", publishHandler).Methods("POST")
    r.HandleFunc("/play/{app_name}/{stream_key}", playHandler)

    srv := &http.Server{
        Addr:         "0.0.0.0:8080",
        // Good practice to set timeouts to avoid Slowloris attacks.
        WriteTimeout: time.Second * 0,
        ReadTimeout:  time.Second * 0,
        IdleTimeout:  time.Second * 60,
        Handler: r, // Pass our instance of gorilla/mux in.
    }

    // Run our server in a goroutine so that it doesn't block.
    go func() {
        if err := srv.ListenAndServe(); err != nil {
            logging.Error("server listen error:", err)
        }
    }()

    c := make(chan os.Signal, 1)
    // We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
    // SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
    signal.Notify(c, os.Interrupt)

    // Block until we receive our signal.
    <-c

    // Create a deadline to wait for.
    ctx, cancel := context.WithTimeout(context.Background(), wait)
    defer cancel()
    // Doesn't block if no connections, but will otherwise wait
    // until the timeout deadline.
    srv.Shutdown(ctx)
    // Optionally, you could run srv.Shutdown in a goroutine and block on
    // <-ctx.Done() if your application should wait for other services
    // to finalize based on context cancellation.
    logging.Info("shutting down")
    os.Exit(0)


    // newclients = make(chan *WsClient, 16)
    // bufCh = make(chan []byte, 1)
    // go broacast()

    // http.HandleFunc("/stream", wsHandler)

    // secretUrl := "/" + *secret
    // http.HandleFunc(secretUrl, streamHandler)

    // //   http.HandleFunc("/", pageHandler)
    // http.Handle("/", http.FileServer(http.Dir("./")))

    // err := http.ListenAndServe(*localAddr, nil)
    // if err != nil {
    //  logging.Error("server listen error:", err)
    // }
}
