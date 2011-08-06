package heckleTypes

import (
     "time"
)

type Ctlmsg struct {
     Addresses []string
     Time      int64
     Image     string
     Extra     map[string]string
}

type ResourceInfo struct {
     Allocated                          bool
     TimeAllocated, AllocationEndTime   int64
     Owner, Image, Comments             string
     AllocationNumber                   uint64
}

type CurrentRequestsNode struct {
     User                string
     Image               string
     Status              string
     AllocationNumber    uint64
     ActivityTimeout     int64
     TryOnFail           bool
     LastActivity        int64
     Info                []InfoMsg
}

type Listmsg struct {
     Addresses           []string
     Image               string
     ActivityTimeout     int64
     AllocNum            int
}

type Nummsg struct {
     NumNodes            int
     Image               string
     ActivityTimeout     int64
}

type StatusMessage struct {
     Status         string
     LastActivity   int64
     Info           []InfoMsg
}

type InfoMsg struct {
     Time    int64
     Message string
     MsgType string
}

type OutletNode struct{
     Address   string
     Outlet    string
}

func (resource *ResourceInfo) Reset() {
     resource.Allocated = false
     resource.TimeAllocated = 0
     resource.AllocationEndTime = 0
     resource.Owner = "None"
     resource.Image = "None"
     resource.Comments = ""
     resource.AllocationNumber = 0
}

func (resource *ResourceInfo) Allocate(owner string, image string, allocationNum uint64) {
     resource.Allocated = true
     resource.Owner = owner
     resource.Image = image
     resource.TimeAllocated = time.Seconds()
     resource.AllocationEndTime = time.Seconds() + 604800
     resource.AllocationNumber = allocationNum
}

func (resource *ResourceInfo) Broken() {
     resource.Allocated = true
     resource.TimeAllocated = time.Seconds()
     resource.AllocationEndTime = 9223372036854775807
     resource.Owner = "System Admin"
     resource.Image = "brokenNode-headAche-amd64"
     resource.Comments = "Installation failed or there was a timeout."
     resource.AllocationNumber = 0
}