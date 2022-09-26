package profile

import (
	"encoding/json"

	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/sample"
)

type (
	profileInterface interface {
		GetID() string
		GetOrganizationID() uint64
		GetProjectID() uint64

		CallTrees() (map[uint64][]*nodetree.Node, error)
		StoragePath() string

		UnmarshalJSON(b []byte) error
	}

	Profile struct {
		version string

		profile profileInterface

		sample *sample.SampleProfile
		legacy *LegacyProfile
	}

	version struct {
		Version string `json:"version"`
	}
)

func (p *Profile) UnmarshalJSON(b []byte) error {
	var v version
	err := json.Unmarshal(b, &v)
	if err != nil {
		return err
	}
	p.version = v.Version
	switch p.version {
	case "":
		p.legacy = new(LegacyProfile)
		p.profile = p.legacy
		return json.Unmarshal(b, &p.legacy)
	default:
		p.sample = new(sample.SampleProfile)
		p.profile = p.sample
		return json.Unmarshal(b, &p.sample)
	}
}

func (p Profile) MarshalJSON() ([]byte, error) {
	switch p.version {
	case "":
		return json.Marshal(p.legacy)
	default:
		return json.Marshal(p.sample)
	}
}

func (p *Profile) CallTrees() (map[uint64][]*nodetree.Node, error) {
	return p.profile.CallTrees()
}

func (p *Profile) GetID() string {
	return p.profile.GetID()
}

func (p *Profile) GetOrganizationID() uint64 {
	return p.profile.GetOrganizationID()
}

func (p *Profile) GetProjectID() uint64 {
	return p.profile.GetProjectID()
}

func (p *Profile) StoragePath() string {
	return p.profile.StoragePath()
}
