package main

import (
  "fmt"
  "os"
  "time"
  "github.com/jessevdk/go-flags"
  "encoding/json"
  "net"
  "net/http"
  "bytes"
  "io/ioutil"
)

type options struct {
  Help         bool      `short:"h" long:"help" description:"show help message"`
  Host         string    `long:"host" default:"" description:"Hostname"`
  Timeout      float64   `short:"i" long:"timeout" default:"1.0" description:"Provide the timeout of the Maintenance Mode action as a float in hours.'"`
  Enable       bool      `short:"e" long:"enable" description:"Enable maintenance mode"`
  Disable      bool      `short:"d" long:"disable" description:"Disable maintenance mode"`
  DisableHost  bool      `short:"a" long:"disableall" description:"Disable all maintenances for host"`
  GetStatus    bool      `short:"g" long:"getstatus" description:"Get maintenance information for host"`
  Silent       bool      `short:"s" long:"silent" description:"Surpress all output"`
  RPD          int       `long:"rpd" default:"0" decription:"RPD ticket number"`
  ID           string    `long:"id" description:"Unique ID returned when the maintenance was created"`
  Status       string    `long:"status" default:"active" description:"Status [active|completed|scheduled|deleted]"`
  ConfigFile   string    `short:"f" long:"file" default:"/etc/fds/icinga.json" description:"Custom config file"`
}

type INI struct {
  BaseURL      string    `json:"BaseURL"`
  APIKEY       string    `json:"API-KEY"`
  Owners       string    `json:"Owners"`
}

type DT struct {
  startTime    string
  endTime      string
}

type MAINT struct {
  Name         string    `json:"name"`
  Hosts        []string  `json:"hosts"`
  AllServices  bool      `json:"allservices"`
  StartTime    string    `json:"startTime"`
  EndTime      string    `json:"endTime"`
  Owners       []string  `json:"owners"`
  Comment      string    `json:"comment"`
  RPD          int       `json:rpd"`
}


type RESPONSE struct {
  MaintenanceId   string    `json:"maintenanceId"`
  Name            string    `json:"name"`
  Type            string    `json:"type"`
  Hosts           []string  `json:"hosts"`
  AllServices     bool      `json:"allServices"`
  StartTime       string    `json:"startTime"`
  EndTime         string    `json:"endTime"`
  CreatedBy       string    `json:"createdBy"`
  CreationTime    string    `json:"creationTime"`
  UpdatedBy       string    `json:"updatedBy"`
  UpdationTime    string    `json:"updationTime"`
  Status          string    `json:"status"`
  Comment         string    `json:"comment"`
  Rpd             int       `json:"rpd"`
}

// --- read config json file ---
func readINI(file string) INI {
    var ini INI
  
  // --- read json ini file ---
  jsonFile, err := os.Open(file)
  if err != nil {
    fmt.Printf("Cannot open config file %s - %s\n", file, err.Error())
    os.Exit(3)
  }
  defer jsonFile.Close()
  
  content, _ := ioutil.ReadAll(jsonFile)
  err = json.Unmarshal(content, &ini)
  if err != nil {
    fmt.Printf("Parse json failed - %s\n", err.Error())
    os.Exit(3)
  }
  return ini
}

// --- check if host is valid (DNS only) ---
func checkHost(host string) bool {
  //dnsHost := fmt.Sprintf("%s.factset.com", host)
  iprecs, err := net.LookupIP(host)
  
  if err != nil || len(iprecs) == 0 {
    return(false)
  } else {
    return(true)
  }
}

// --- get current start and end times ---
func getDateTime(timeout float64) DT {
  ts := time.Now()
  te := ts.Add(time.Second * time.Duration(timeout * 3600))
  dt := DT { 
    ts.Format(time.RFC3339),
    te.Format(time.RFC3339),
  }
  return dt
}

// --- enable maintenacse mode ---
func maint_enable(opts options, ini INI) {
  // -- check host --
  if !checkHost(opts.Host) {
    if !opts.Silent {
      fmt.Printf("Host: %s not found!\n", opts.Host)
    }
    os.Exit(-1)
  }

  dt := getDateTime(opts.Timeout)

  // -- prepare json --
  maint := MAINT {
    opts.Host,
    []string{opts.Host},
    true,
    dt.startTime,
    dt.endTime,
    []string{ini.Owners},
    "Automatic maintenance mode set by " + ini.Owners,
    opts.RPD,
  }
  
  e, err := json.Marshal(maint)
  if err != nil {
    fmt.Println(err)
    return
  }
  if !opts.Silent {
    fmt.Println(string(e))
  }

  url  := fmt.Sprintf("%shost", ini.BaseURL)
  auth := fmt.Sprintf("API-KEY %s", ini.APIKEY)
  
  body := bytes.NewReader(e)
  req, err := http.NewRequest("POST", url, body)
  if err != nil {
    panic(err.Error())
  }
  req.Header.Set("Content-Type", "application/json")
  req.Header.Set("Authorization", auth)
  resp, err := http.DefaultClient.Do(req)
  if err != nil {
    panic(err.Error())
  }
  defer resp.Body.Close()    

  bodyBytes, err := ioutil.ReadAll(resp.Body)
  
  if !opts.Silent {
    fmt.Println(string(bodyBytes))
  }
  
  os.Exit(0)
}

// --- disable (delete) maintenacse mode ---
func maint_disable(opts options, ini INI) {
  var str       []byte

  // -- verify if maintenence ID provided --
  if opts.ID == "" {
    if !opts.Silent {
      fmt.Println("Maintenance id must be provided for deletion!")
    }
    os.Exit(3)
  }
    
  // -- prepare command line for curl --
  url  := fmt.Sprintf("%s%s", ini.BaseURL, opts.ID)
  auth := fmt.Sprintf("API-KEY %s", ini.APIKEY)

  // -- excute --
  body := bytes.NewReader(str)
  req, err := http.NewRequest("DELETE", url, body)
  if err != nil {
    panic(err.Error())
  }
  req.Header.Set("Content-Type", "application/json")
  req.Header.Set("Authorization", auth)
  resp, err := http.DefaultClient.Do(req)
  if err != nil {
    panic(err.Error())
  }
  defer resp.Body.Close()    

  bodyBytes, err := ioutil.ReadAll(resp.Body)

  if !opts.Silent {
    fmt.Println(string(bodyBytes))
  }
    
  os.Exit(0)
}

// --- disable (delete) all maintenacse for host ---
func maint_disableHost(opts options, ini INI) {
  var str       []byte

  // -- verify if provided host is valid (DNS) --
  if !checkHost(opts.Host) {
    if !opts.Silent {
      fmt.Printf("Host: %s not found!\n", opts.Host)
    }
    os.Exit(3)
  }
    
  // -- prepare command line for curl request --
  url  := fmt.Sprintf("%shost/%s", ini.BaseURL, opts.Host)
  auth := fmt.Sprintf("API-KEY %s", ini.APIKEY)

  // -- excute --
  body := bytes.NewReader(str)
  req, err := http.NewRequest("DELETE", url, body)
  if err != nil {
    panic(err.Error())
  }
  req.Header.Set("Content-Type", "application/json")
  req.Header.Set("Authorization", auth)
  resp, err := http.DefaultClient.Do(req)
  if err != nil {
    panic(err.Error())
  }
  defer resp.Body.Close()    

  bodyBytes, err := ioutil.ReadAll(resp.Body)

  if !opts.Silent {
    fmt.Println(string(bodyBytes))
  }
  
  os.Exit(0)
}

// --- get maintenance information for host ---
func maint_get(opts options, ini INI) {
  var str       []byte
  var response  []RESPONSE
  
  // -- check host --
  if !checkHost(opts.Host) {
    if !opts.Silent {
      fmt.Printf("Host: %s not found!\n", opts.Host)
    }
    os.Exit(3)
  }

  // -- prepare url for curl --
  url  := fmt.Sprintf("%shost/all/%s?status=%s", ini.BaseURL, opts.Host, opts.Status)
  auth := fmt.Sprintf("API-KEY %s", ini.APIKEY)

  body := bytes.NewReader(str)
  req, err := http.NewRequest("GET", url, body)
  if err != nil {
    panic(err.Error())
  }
  req.Header.Set("Content-Type", "application/json")
  req.Header.Set("Authorization", auth)
  resp, err := http.DefaultClient.Do(req)
  if err != nil {
    panic(err.Error())
  }
  defer resp.Body.Close()    

  // --- parse response ---
  bodyBytes, _ := ioutil.ReadAll(resp.Body)
  err = json.Unmarshal(bodyBytes, &response)
  if err != nil {
    panic(err.Error())
  }
  
  if !opts.Silent {
    for i, resp := range response {
      serv := "false"
      if resp.AllServices {
        serv = "true"
      }
      fmt.Printf("\n ------------- Maintenance #%d -------------\n", i+1)
      fmt.Printf("nmaintenanceId: %s\n", resp.MaintenanceId)
      fmt.Printf("name: %s\n", resp.Name)
      fmt.Printf("type: %s\n", resp.Type)
      fmt.Printf("hosts: %s\n", resp.Hosts[0])
      fmt.Printf("allServices: %s\n", serv)
      fmt.Printf("startTime: %s\n", resp.StartTime)
      fmt.Printf("endTime: %s\n", resp.EndTime)
      fmt.Printf("createdBy: %s\n", resp.CreatedBy)
      fmt.Printf("creationTime: %s\n", resp.CreationTime)
      fmt.Printf("updatedBy: %s\n", resp.UpdatedBy)
      fmt.Printf("updationTime: %s\n", resp.UpdationTime)
      fmt.Printf("status: %s\n", resp.Status)
      fmt.Printf("comment: %s\n", resp.Comment)
      fmt.Printf("rpd: %d\n", resp.Rpd)
    }
  }
  
  if len(response) > 0 {
    os.Exit(0)
  } else {
    os.Exit(1)
  }
}

func main() {
  var opts options
  
  // --- parse commant line arguments ---
  p := flags.NewParser(&opts, flags.Default&^flags.HelpFlag)
  _, err := p.Parse()
  if err != nil {
    fmt.Printf("Fail to parse args: %v", err)
    os.Exit(3)
  }

  if opts.Help {
    p.WriteHelp(os.Stdout)
    os.Exit(0)
  }

  // --- get settings from config file ---
  ini := readINI(opts.ConfigFile)

  // --- validate arguments ---
  if opts.Host == ""  && (opts.Enable || opts.GetStatus || opts.DisableHost) {
    p.WriteHelp(os.Stdout)
    os.Exit(3)
  }
  if opts.GetStatus && opts.Status != "active" && opts.Status != "completed" && opts.Status != "scheduled" && opts.Status != "deleted" {
    p.WriteHelp(os.Stdout)
    os.Exit(3)
  }
    
  if opts.Enable {
    maint_enable(opts, ini)
  }
  
  if opts.Disable {
    maint_disable(opts, ini)
  }

  if opts.DisableHost {
    maint_disableHost(opts, ini)
  }

  if opts.GetStatus {
    maint_get(opts, ini)
  }
  
  os.Exit(0)
}
