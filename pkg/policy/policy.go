// Copyright 2016-2017 Authors of Cilium
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package policy

import (
	"crypto/sha512"
	"fmt"
	"strconv"
	"strings"

	"github.com/cilium/cilium/api/v1/models"
	"github.com/cilium/cilium/pkg/labels"

	"github.com/op/go-logging"
)

var (
	log = logging.MustGetLogger("cilium-policy")
)

// Privilege represents the privileges available to define for a policy node.
type Privilege byte

const (
	ALLOW Privilege = iota
	ALWAYS_ALLOW
	REQUIRES
	L4
)

var (
	privEnc = map[Privilege]string{
		ALLOW:        "allow",
		ALWAYS_ALLOW: "always-allow",
		REQUIRES:     "requires",
		L4:           "l4",
	}
	privDec = map[string]Privilege{
		"allow":        ALLOW,
		"always-allow": ALWAYS_ALLOW,
		"requires":     REQUIRES,
		"l4":           L4,
	}
)

func (p Privilege) String() string {
	if v, exists := privEnc[p]; exists {
		return v
	}
	return ""
}

func (p *Privilege) UnmarshalJSON(b []byte) error {
	if p == nil {
		p = new(Privilege)
	}
	if len(b) <= len(`""`) {
		return fmt.Errorf("invalid privilege '%s'", string(b))
	}
	if v, exists := privDec[string(b[1:len(b)-1])]; exists {
		*p = Privilege(v)
		return nil
	}

	return fmt.Errorf("unknown '%s' privilege", string(b))
}

func (p Privilege) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, p)), nil
}

type Tracing int

const (
	TRACE_DISABLED Tracing = iota
	TRACE_ENABLED
	TRACE_VERBOSE
)

// PolicyTrace logs the given message into the SearchContext logger only if
// TRACE_ENABLED or TRACE_VERBOSE is enabled in the receiver's SearchContext.
func (s *SearchContext) PolicyTrace(format string, a ...interface{}) {
	switch s.Trace {
	case TRACE_ENABLED, TRACE_VERBOSE:
		log.Debugf(format, a...)
		if s.Logging != nil {
			format = "%-" + s.CallDepth() + "s" + format
			a = append([]interface{}{""}, a...)
			s.Logging.Logger.Printf(format, a...)
		}
	}
}

// PolicyTraceVerbose logs the given message into the SearchContext logger only
// if TRACE_VERBOSE is enabled in the receiver's SearchContext.
func (s *SearchContext) PolicyTraceVerbose(format string, a ...interface{}) {
	switch s.Trace {
	case TRACE_VERBOSE:
		log.Debugf(format, a...)
		if s.Logging != nil {
			s.Logging.Logger.Printf(format, a...)
		}
	}
}

type SearchContext struct {
	Trace   Tracing
	Depth   int
	Logging *logging.LogBackend
	From    labels.LabelArray
	To      labels.LabelArray
	DPorts  []*models.Port
}

func (s *SearchContext) String() string {
	from := []string{}
	to := []string{}
	dports := []string{}
	for _, fromLabel := range s.From {
		from = append(from, fromLabel.String())
	}
	for _, toLabel := range s.To {
		to = append(to, toLabel.String())
	}
	for _, dport := range s.DPorts {
		dports = append(dports, fmt.Sprintf("%d/%s", dport.Port, dport.Protocol))
	}
	ret := fmt.Sprintf("From: [%s]", strings.Join(from, ", "))
	ret += fmt.Sprintf(" => To: [%s]", strings.Join(to, ", "))
	if len(dports) != 0 {
		ret += fmt.Sprintf(" AND to destination ports: [%s]", strings.Join(dports, ", "))
	}
	return ret
}

func (s *SearchContext) CallDepth() string {
	return strconv.Itoa(s.Depth * 2)
}

// TargetCoveredBy checks if the SearchContext `To` is covered by the all
// `coverage` labels.
func (s *SearchContext) TargetCoveredBy(coverage []*labels.Label) bool {
	s.PolicyTraceVerbose("Checking if %+v covers %+v", coverage, s.To)
	return s.To.Contains(coverage)
}

var (
	CoverageSHASize = len(fmt.Sprintf("%x", sha512.New512_256().Sum(nil)))
)
