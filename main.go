package main

import(
        "encoding/json"
        "fmt"
        "log"
        "net"
        "net/url"
        "net/http"
        "net/http/httputil"

        "github.com/fsnotify/fsnotify"
        "github.com/spf13/viper"
)

var (
        allowedIP string
        revProxURL string
        listenHostPort string

        liveReload = false
        devMode = false

        hostProxy = make(map[string]*httputil.ReverseProxy)
)

type baseHandle struct{}

func headerToJson(header http.Header) {

        jsonHeader, err := json.Marshal(header)
        if err != nil {
                fmt.Println(err)
        }

        log.Println(string(jsonHeader))

}

func startRevProx() {
        
        h := &baseHandle{}
        http.Handle("/", h)

        server := &http.Server {
                Addr: listenHostPort,
                Handler: h,
        }

        log.Printf("Starting server listening on %s", listenHostPort)

        err := server.ListenAndServe()

        if err != nil {
                panic(err)
        }
}

func (h *baseHandle) ServeHTTP(w http.ResponseWriter, r *http.Request) {

        rip := r.Header.Get("X-Forwarded-For")
        if rip == "" {
                var err error 
                rip, _, err = net.SplitHostPort(r.RemoteAddr)
                if err != nil {
                        log.Println(err)
                }
        }

        r.Header.Add("URL", r.URL.String())
        headerToJson(r.Header)

        if rip == allowedIP || devMode {
                remoteURL, err := url.Parse(revProxURL)
                if err != nil {
                        log.Println(err)
                }
                proxy := httputil.NewSingleHostReverseProxy(remoteURL)

                w.Header().Set("X-Server", "revprox")
                proxy.ServeHTTP(w, r)

        } else {

                http.Error(w, "Not authorized", http.StatusUnauthorized)

        }
        
}

func loadConfig() {

        viper.SetConfigName("config")
        viper.SetConfigType("yaml")
        viper.AddConfigPath("/etc/revprox/")
        viper.AddConfigPath(".") 
        
        err := viper.ReadInConfig()
        if err != nil {
                panic(fmt.Errorf("Fatal error config file: %s \n", err))
        }

        allowedIP = viper.GetString("allowedIP")
        revProxURL = viper.GetString("revProxURL")
        listenHostPort = viper.GetString("listenHostPort")
        liveReload = viper.GetBool("liveReload")
        devMode = viper.GetBool("devMode")

        if liveReload {
        
                viper.WatchConfig()
                viper.OnConfigChange(func(e fsnotify.Event) {
                        log.Println("Config file changed:", e.Name)
                        allowedIP = viper.GetString("allowedIP")
                        revProxURL = viper.GetString("revProxURL")
                        devMode = viper.GetBool("devMode")
                })
        
        }
        
}

func main() {

        loadConfig()
        startRevProx()

}
