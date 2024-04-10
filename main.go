package main

import (
  "bytes"
  "encoding/json"
  "os"
  "io/ioutil"
  "time"
  "net/http"
  "strconv"
  "strings"

  logger "zhifou/utils/logger"
  parser "zhifou/utils/parser"

  "github.com/shirou/gopsutil/v3/cpu"
  "github.com/shirou/gopsutil/v3/mem"
  "github.com/shirou/gopsutil/v3/disk"
  "github.com/shirou/gopsutil/v3/net"
  "github.com/shirou/gopsutil/v3/host"
  "github.com/shirou/gopsutil/v3/load"
)

var lastNicIOMap map[string]net.IOCountersStat
var lastDiskIOMap map[string]disk.IOCountersStat

var server = os.Getenv("SERVER_URL")
var interval, _ = strconv.Atoi(os.Getenv("INTERVAL"))
var u64itv = uint64(interval)
var clientID = os.Getenv("CLIENT_ID")
var rootfs = os.Getenv("HOST_ROOT")

type Host struct {
  ClientID             string `json:"client_id"`
  Hostname             string `json:"hostname"`
  Uptime               uint64 `json:"uptime"`
  BootTime             uint64 `json:"bootTime"`
  Procs                uint64 `json:"procs"`
  OS                   string `json:"os"`
  Platform             string `json:"platform"`
  PlatformFamily       string `json:"platformFamily"`
  PlatformVersion      string `json:"platformVersion"`
  KernelVersion        string `json:"kernelVersion"`
  KernelArch           string `json:"kernelArch"`
  VirtualizationSystem string `json:"virtualizationSystem"`
  VirtualizationRole   string `json:"virtualizationRole"`
  HostID               string `json:"hostid"`
}

type Partition struct {
  Device          string            `json:"device"`
  MountPoint      string            `json:"mount_point"`
  FSType          string            `json:"fs_type"`
  Used            uint64            `json:"used"`
  Free            uint64            `json:"free"`
  Total           uint64            `json:"total"`
  Percent         float64           `json:"percent"`
  ReadBps         uint64            `json:"read_bps"`
  WriteBps        uint64            `json:"write_bps"`
  ReadIOPS        uint64            `json:"read_iops"`
  WriteIOPS       uint64            `json:"write_iops"`
}

type Interface struct {
  Name            string            `json:"name"`
  MAC             string            `json:"mac"`
  Addr            string            `json:"addr"`
  RecvBytes       uint64            `json:"recv_bytes"`
  SentBytes       uint64            `json:"sent_bytes"`
  RecvPackets     uint64            `json:"recv_packets"`
  SentPackets     uint64            `json:"sent_packets"`
  RecvBps         uint64            `json:"recv_bps"`
  SentBps         uint64            `json:"sent_bps"`
  RecvPps         uint64            `json:"recv_pps"`
  SentPps         uint64            `json:"sent_pps"`
}

type Connection struct {
  Established       uint64            `json:"established"`
  SynSent           uint64            `json:"syn_sent"`
  SynRecv           uint64            `json:"syn_recv"`
  FinWait1          uint64            `json:"fin_wait1"`
  FinWait2          uint64            `json:"fin_wait2"`
  TimeWait          uint64            `json:"time_wait"`
  Close             uint64            `json:"close"`
  CloseWait         uint64            `json:"close_wait"`
  LastAck           uint64            `json:"last_ack"`
  Listen            uint64            `json:"listen"`
  Closing           uint64            `json:"closing"`
}

type Info struct {
  ClientID      string        `json:"client_id"`
  Connection    Connection    `json:"connection"`
  CPU           float64       `json:"cpu"`
  CPUPerCore    []float64     `json:"cpu_per_core"`
  Disk          []Partition   `json:"disk"`
  Load1         float64       `json:"load1"`
  Load15        float64       `json:"load15"`
  Load5         float64       `json:"load5"`
  MEM           float64       `json:"mem"`
  MEMUsed       uint64        `json:"mem_used"`
  MEMFree       uint64        `json:"mem_free"`
  MEMTotal      uint64        `json:"mem_total"`
  Network       []Interface   `json:"network"`
  ProcessCount  uint64        `json:"process_count"`
  Swap          float64       `json:"swap"`
  SwapUsed      uint64        `json:"swap_used"`
  SwapFree      uint64        `json:"swap_free"`
  SwapTotal     uint64        `json:"swap_total"`
}

func postMetrics() {
  var info Info
  info.ClientID = clientID

  // cpu
  info.CPUPerCore, _ = cpu.Percent(0, true)
  cpuinfo, _ := cpu.Percent(0, false)
  for i := 0; i < len(info.CPUPerCore); i ++ {
    info.CPUPerCore[i] = parser.ParseFloat(info.CPUPerCore[i], 3)
  }
  info.CPU = parser.ParseFloat(cpuinfo[0], 3)
  
  // load
  _load, _ := load.Avg()
  info.Load1 = _load.Load1
  info.Load5 = _load.Load5
  info.Load15 = _load.Load15

  // mem
  meminfo, _ := mem.VirtualMemory()
  info.MEM = parser.ParseFloat(meminfo.UsedPercent, 3)
  info.MEMFree = meminfo.Free
  info.MEMUsed = meminfo.Used
  info.MEMTotal = meminfo.Total

  // swap
  swapinfo, _ := mem.SwapMemory()
  info.Swap = parser.ParseFloat(swapinfo.UsedPercent, 3)
  info.SwapFree = swapinfo.Free
  info.SwapUsed = swapinfo.Used
  info.SwapTotal = swapinfo.Total
  
  // process count
  hostInfo, _ := host.Info()
  info.ProcessCount = hostInfo.Procs
  
  // disk info
  partitions, _ := disk.Partitions(false)
  diskIOMap := make(map[string]disk.IOCountersStat)
  var diskinfo []Partition
  for _, v := range partitions {
    var part Partition
    part.Device = v.Device
    part.MountPoint = v.Mountpoint
    part.FSType = v.Fstype
    usageStat, err := disk.Usage(rootfs + part.MountPoint)
    if (err != nil) {
      logger.Error(v.String())
      logger.Error(err.Error())
    }
    part.Used = usageStat.Used
    part.Free = usageStat.Free
    part.Percent = parser.ParseFloat(usageStat.UsedPercent,3 )
    part.Total = usageStat.Total
    ioStat, _ := disk.IOCounters(v.Device)
    for _, diskIO := range ioStat {
      diskIOMap[v.Device] = diskIO
      lastDiskIO, lastDiskIOExists := lastDiskIOMap[v.Device]
      if (lastDiskIOExists) {
        if (diskIO.ReadBytes >= lastDiskIO.ReadBytes) {
          part.ReadBps = uint64((diskIO.ReadBytes - lastDiskIO.ReadBytes) / u64itv)
        }
        if (diskIO.WriteBytes >= lastDiskIO.WriteBytes) {
          part.WriteBps = uint64((diskIO.WriteBytes - lastDiskIO.WriteBytes) / u64itv)
        }
        if (diskIO.ReadCount >= lastDiskIO.ReadCount) {
          part.ReadIOPS = uint64((diskIO.ReadCount - lastDiskIO.ReadCount) / u64itv)
        }
        if (diskIO.WriteCount >= lastDiskIO.WriteCount) {
          part.WriteIOPS = uint64((diskIO.WriteCount - lastDiskIO.WriteCount) / u64itv)
        }
      }
    }
    lastDiskIOMap = diskIOMap
    diskinfo = append(diskinfo, part)
  }
  info.Disk = diskinfo

  // connections
  connections, _ := net.Connections("tcp")
  var conncetionStat Connection
  for _, conn := range connections {
    switch conn.Status {
      case "ESTABLISHED": conncetionStat.Established++
      case "SYN_SENT": conncetionStat.SynSent++
      case "SYN_RECV": conncetionStat.SynRecv++
      case "FIN_WAIT1": conncetionStat.FinWait1++
      case "FIN_WAIT2": conncetionStat.FinWait2++
      case "TIME_WAIT": conncetionStat.TimeWait++
      case "CLOSE": conncetionStat.Close++
      case "CLOSE_WAIT": conncetionStat.CloseWait++
      case "LAST_ACK": conncetionStat.LastAck++
      case "LISTEN": conncetionStat.Listen++
      case "CLOSING": conncetionStat.Closing++
    }
  }
  info.Connection = conncetionStat

  // net info
  interfaces, _ := net.Interfaces()
  netstats, _ := net.IOCounters(true)
  nicIOMap := make(map[string]net.IOCountersStat)
  for _, netstat := range netstats {
    nicIOMap[netstat.Name] = netstat
  }
  var netinfo []Interface
  for _, v := range interfaces {
    if v.Name == "lo" ||
      strings.HasPrefix(v.Name, "veth") ||
      strings.HasPrefix(v.Name, "br-") ||
      strings.HasPrefix(v.Name, "docker") {
      continue
    }
    // nic base info
    var _interface Interface
    _interface.Name = v.Name
    _interface.MAC = v.HardwareAddr
    for _, addr := range v.Addrs {
      if strings.Contains(addr.Addr, ":") {
        continue
      }
      _interface.Addr = addr.Addr
    }
    // nic io info
    nicIO, nicIOExists := nicIOMap[v.Name]
    lastNicIO, lastNicIOExists := lastNicIOMap[v.Name]
    if (nicIOExists) {
      _interface.RecvBytes = nicIO.BytesRecv
      _interface.SentBytes = nicIO.BytesSent
      _interface.RecvPackets = nicIO.PacketsRecv
      _interface.SentPackets = nicIO.PacketsSent
    }
    if (lastNicIOExists) {
      _interface.RecvBps = uint64((nicIO.BytesRecv - lastNicIO.BytesRecv) / u64itv)
      _interface.SentBps = uint64((nicIO.BytesSent - lastNicIO.BytesSent) / u64itv)
      _interface.RecvPps = uint64((nicIO.PacketsRecv - lastNicIO.PacketsRecv) / u64itv)
      _interface.SentPps = uint64((nicIO.PacketsSent - lastNicIO.PacketsSent) / u64itv)
    }
    netinfo = append(netinfo, _interface)
  }
  lastNicIOMap = nicIOMap
  info.Network = netinfo

  body, _ := json.Marshal(info)
  resp, err := http.Post(server + "/metrics/post", "application/json", bytes.NewBuffer([]byte(body)))
  if err != nil {
    // fmt.Println(err.Error())
    logger.Error(err.Error())
    return
  }
  result, _ := ioutil.ReadAll(resp.Body)
  if !strings.Contains(string(result), "ok") {
    logger.Warn("incorrect return message: " + string(result))
  }
}

func postHostInfo() {
  // host
  hostInfo, _ := host.Info()
  var _host Host
  _host.ClientID = clientID
  _host.Hostname = hostInfo.Hostname
  _host.Uptime = hostInfo.Uptime
  _host.BootTime = hostInfo.BootTime
  _host.Procs = hostInfo.Procs
  _host.OS = hostInfo.OS
  _host.Platform = hostInfo.Platform
  _host.PlatformFamily = hostInfo.PlatformFamily
  _host.PlatformVersion = hostInfo.PlatformVersion
  _host.KernelVersion = hostInfo.KernelVersion
  _host.KernelArch = hostInfo.KernelArch
  _host.VirtualizationSystem = hostInfo.VirtualizationSystem
  _host.VirtualizationRole = hostInfo.VirtualizationRole
  _host.HostID = hostInfo.HostID

  body, _ := json.Marshal(_host)
  resp, err := http.Post(server + "/metrics/hostinfo", "application/json", bytes.NewBuffer([]byte(body)))
  if err != nil {
    // fmt.Println(err.Error())
    logger.Error(err.Error())
    return
  }
  result, _ := ioutil.ReadAll(resp.Body)
  if !strings.Contains(string(result), "ok") {
    logger.Warn("incorrect return message: " + string(result))
  }
}

func main() {
  postHostInfo();
  for range time.Tick(time.Second * time.Duration(interval)) {
    postMetrics()
  }
}