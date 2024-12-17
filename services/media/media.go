package media

import (
	"fmt"
	"io"

	"github.com/gofiber/fiber/v2/log"
)

type AcceptedObjContentType uint8

const (
	PDF AcceptedObjContentType = iota
	IMG
	_ContentTypeCounter
)

type MediaStoreFolderName string

const (
	InpDocumentsFolder   = "textbased"
	InImagesFolder       = "images"
	OutDocSummaryFolder  = "file_summarys"
	OutImgAnalysisFolder = "img_analysis"
	OutQuizGenFolder     = "quizzs"
	OutTestsFolder       = "tests"
)

var (
	InputFolders = []string{
		InImagesFolder,
		InpDocumentsFolder,
	}

	OutputFolders = []string{
		OutQuizGenFolder,
		OutImgAnalysisFolder,
		OutDocSummaryFolder,
		OutTestsFolder,
	}
)

func MapContentTypeToFolder(contentType AcceptedObjContentType) (string, error) {
	if contentType > _ContentTypeCounter {
		return "", fmt.Errorf("Invalid value for content type")
	}

	switch contentType {
	case PDF:
		return InpDocumentsFolder, nil
	case IMG:
		return InImagesFolder, nil
	default:
		return "", fmt.Errorf("Bad content type passed as argument")
	}
}

type UserFileUploadReq struct {
	bucket        string
	folder        string
	objName       string
	contentType   AcceptedObjContentType
	contentReader io.Reader
}

func NewUserFileUpRequest(
	bucket string,
	folder string,
	objName string,
	contentType AcceptedObjContentType,
	contentREader io.Reader,
) UserFileUploadReq {
	return UserFileUploadReq{
		bucket:        bucket,
		folder:        folder,
		objName:       objName,
		contentType:   contentType,
		contentReader: contentREader,
	}
}

func (u UserFileUploadReq) Bucket() string {
	return u.bucket
}

func (u UserFileUploadReq) Folder() string {
	return u.folder
}

func (u UserFileUploadReq) ObjName() string {
	return u.objName
}

func (u UserFileUploadReq) ContentType() string {
	contentType, err := MapContentTypeToFolder(u.contentType)
	if err != nil {
		log.Panic("Should not reach this part! This impl should always return a valid value")
	}
	return contentType
}

func (u UserFileUploadReq) ContentReader() io.Reader {
	return u.contentReader
}
