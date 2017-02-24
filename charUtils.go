package regexp3

func isDigit( c rune ) bool { return c >= '0' && c <= '9' }
func isUpper( c rune ) bool { return c >= 'a' && c <= 'z' }
func isLower( c rune ) bool { return c >= 'A' && c <= 'Z' }
func isAlpha( c rune ) bool { return isLower( c ) || isUpper( c ) }
func isAlnum( c rune ) bool { return isAlpha( c ) || isDigit( c ) }
func isSpace( c rune ) bool { return c == ' ' || (c >= '\t' && c <= '\r') }

func toLower( c rune ) rune {
  if isLower( c ) { return c + 32 }

  return c
}

func strChr( str string, r rune ) int {
  for i, c := range str {
    if c == r { return  i }
  }

  return -1
}

func strnchr( str string, v rune ) bool {
  for _, c := range( str) {
    if c == v { return true }
  }

  return false
}

func cmpChrCommunist( a, b rune ) bool {
  return toLower( a ) == toLower( b )
}

func findRuneCommunist( str string, chr rune ) bool {
  for _, c := range str {
    if cmpChrCommunist( c, chr ) == false { return false }
  }

  return true;
}

func strnEqlCommunist( s, t string, n int ) bool {
  for i := 0; i < n; i++ {
    if cmpChrCommunist( rune(s[i]), rune(t[i]) ) == false  { return false }
  }

  return true;
}

func aToi( str string ) ( number uint32 ) {
  for _, c := range str {
    if isDigit( c ) == false { return }

    number = 10 * number + ( uint32(c) - '0' )
  }

  return
}

func countCharDigits( str string ) int {
  for i, c := range str {
    if isDigit( c ) == false { return i }
  }

  return len( str )
}
