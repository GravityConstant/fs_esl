package dialplan

func init() {
	HandleFunc("/call/", call)
	HandleFunc("/ivr/:first/", ivr)
	HandleFunc("/busy/", busy)
}
