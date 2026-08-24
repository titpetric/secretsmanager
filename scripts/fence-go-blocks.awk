# Fences the code blocks go-fsck copies out of doc comments.
#
# A doc comment writes a code block by indenting it, which markdown renders
# as code without saying what language it is. Every one of them is Go, so
# each run of indented lines outside an existing fence becomes a ```go
# block with one level of indentation removed.
#
# Usage: awk -f scripts/fence-go-blocks.awk docs/api.md

# flush writes the buffered lines as a fenced block, keeping the blank lines
# which followed it outside the fence.
function flush(   i, line, blanks) {
	if (count == 0) {
		return
	}

	for (blanks = 0; count > 0 && buffer[count] == ""; count--) {
		blanks++
	}

	print "```go"
	for (i = 1; i <= count; i++) {
		line = buffer[i]
		sub(/^\t/, "", line)
		print line
	}
	print "```"

	for (i = 0; i < blanks; i++) {
		print ""
	}

	count = 0
}

/^```/ {
	flush()
	fenced = !fenced
	print
	next
}

fenced {
	print
	next
}

/^\t/ {
	buffer[++count] = $0
	next
}

# A blank line ends a block only if prose follows it, so it's held until the
# next line says which.
/^$/ {
	if (count > 0) {
		buffer[++count] = $0
		next
	}
	print
	next
}

{
	flush()
	print
}

END {
	flush()
}
