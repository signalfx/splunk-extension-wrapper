package extensionapi

type ApiError string

func (err ApiError) Error() string {
	return string(err)
}
