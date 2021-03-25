package main

import(
        "encoding/json"
        "fmt"
        "log"
        "net/url"
        "net/http"
        "net/http/httputil"

        "github.com/fsnotify/fsnotify"
        "github.com/spf13/viper"
)

var (
        allowedIP string
        revProxURL string
        listenPort string

        quit = make(chan bool)
        liveReload = false
)

func headerToJson(header http.Header) {
        jsonHeader, err := json.Marshal(header)
        if err != nil {
                fmt.Println(err)
        }
        log.Println(string(jsonHeader))
}

func startRevProx() {
        remote, err := url.Parse(revProxURL)
        if err != nil {
                panic(err)
        } 

        proxy := httputil.NewSingleHostReverseProxy(remote)
        http.HandleFunc("/", handler(proxy))
        log.Printf("Starting server listening on %s", listenPort)
        err = http.ListenAndServe(listenPort, nil)
        if err != nil {
                panic(err)
        }
}

func handler(p *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
        return func(w http.ResponseWriter, r *http.Request) {
                rip := r.Header.Get("X-Forwarded-For")
                r.Header.Add("URL", r.URL.String())
                headerToJson(r.Header)

                if rip != allowedIP {
                        http.Error(w, "Not authorized", http.StatusUnauthorized)
                } else {
                        w.Header().Set("X-Server", "revprox")
                        p.ServeHTTP(w, r)
                }
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
        listenPort = viper.GetString("listenPort")
        liveReload = viper.GetBool("liveReload")

        if liveReload {
                viper.WatchConfig()
                viper.OnConfigChange(func(e fsnotify.Event) {
                        log.Println("Config file changed:", e.Name)
                        allowedIP = viper.GetString("allowedIP")
                })
        }
}

func main() {
        loadConfig()
        startRevProx()
}
