package action

import (
	"encoding/xml"
	"swiflow/ability"
	"swiflow/support"
)

type PathListFiles struct {
	XMLName xml.Name `xml:"path-list-files"`

	Path string `xml:"path" json:"path"`

	Result any `xml:"result" json:"result"`
}

func (act *PathListFiles) Handle(super *SuperAction) any {
	fileAbility := ability.FileSystemAbility{
		Path: act.Path, Base: super.Payload.Home,
	}
	data, err := fileAbility.List()
	if err != nil {
		act.Result = err.Error()
		return err
	}
	header := []string{
		"mode", "user", "group",
		"size", "time", "name",
	}
	act.Result = support.MdTable(data, header)
	return act.Result
}

type FileGetContent struct {
	XMLName xml.Name `xml:"file-get-content"`

	Path string `xml:"path" json:"path"`

	Result any `xml:"result" json:"result"`
}

func (act *FileGetContent) Handle(super *SuperAction) any {
	fileAbility := ability.FileSystemAbility{
		Path: act.Path, Base: super.Payload.Home,
	}
	if v, err := fileAbility.Read(); err != nil {
		act.Result = err.Error()
	} else {
		act.Result = string(v)
	}
	return act.Result
}

type FilePutContent struct {
	XMLName xml.Name `xml:"file-put-content"`

	Path string `xml:"path" json:"path"`
	Data string `xml:"data" json:"data"`

	Result any `xml:"result" json:"result"`
}

func (act *FilePutContent) Handle(super *SuperAction) any {
	if err := super.Payload.InitHome(); err != nil {
		act.Result = err.Error()
		return err
	}

	fileAbility := ability.FileSystemAbility{
		Path: act.Path, Base: super.Payload.Home,
	}
	if err := fileAbility.Write(act.Data); err != nil {
		act.Result = err.Error()
	} else {
		act.Result = act.Data
	}
	return "success"
}

type FileReplaceText struct {
	XMLName xml.Name `xml:"file-replace-text"`

	Path string `xml:"path" json:"path"`
	Diff string `xml:"diff" json:"diff"`

	Result any `xml:"result" json:"result"`
}

func (act *FileReplaceText) Handle(super *SuperAction) any {
	if err := super.Payload.InitHome(); err != nil {
		act.Result = err.Error()
		return err
	}
	fileAbility := ability.FileSystemAbility{
		Path: act.Path, Base: super.Payload.Home,
	}
	if err := fileAbility.Replace(act.Diff); err != nil {
		act.Result = err.Error()
		return err
	} else {
		act.Result = act.Diff
	}
	return "success"
}
