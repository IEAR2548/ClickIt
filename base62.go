package main

const base62Alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func EncodeBase62(id int64) string {
	const offset int64 = 20000000
	id += offset

	if id == 0 {
		return string(base62Alphabet[0])
	}

	base := int64(len(base62Alphabet))
	var encoded []byte
	for id > 0 {
		remainder := id % base
		encoded = append([]byte{base62Alphabet[remainder]}, encoded...)
		id /= base
	}

	return string(encoded)
}
