package routes

type verification struct {
	// signature without 0x prefix is broken into
	// V: sig[0:63]
	// R: sig[64:127]
	// S: sig[128:129]
	Signature string `json:"signature" binding:"required,len=130"`
	Address   string `json:"address" binding:"required,len=42"`
}

func verifyRequest() {

}
