package proto

import "encoding/xml"

type User struct {
	Nick      string
	ShareSize uint64
	FileList  *FileList
}

type File struct {
	Name string `xml:",attr"`
	Size uint64 `xml:",attr"`
	TTH  string `xml:",attr"`
}

type Directory struct {
	Name        string      `xml:",attr"`
	Files       []File      `xml:"File"`
	Directories []Directory `xml:"Directory"`
}

type FileList struct {
	XMLName     xml.Name    `xml:"FileListing"`
	Directories []Directory `xml:"Directory"`
	Files       []File      `xml:"Directory File"`
}
