package regexp3

const INF = 1073741824 // 2^30

const MOD_ALPHA      = 1
const MOD_OMEGA      = 2
const MOD_LONLEY     = 4
const MOD_FwrByChar  = 8
const MOD_COMMUNISM  = 16
const MOD_CAPITALISM = 0xEF

type TEXT struct {
  src string
  init int
  len  int
  pos  int
}

var text TEXT

type CATCH struct {
  init, end int
  id  uint32
}

const MAXCATCH = 16
var   Catch [MAXCATCH]CATCH
var   Cidx, Cindex uint32

const ( PATH = iota; GROUP; HOOK; SET; BACKREF; META; RANGEAB; POINT; SIMPLE )

type RE struct {
  str                string
  kind               uint8
  mods               uint8
  loopsMin, loopsMax uint32
}

func Regexp3( txt, re string ) (result uint) {
  rexp := RE{ re, PATH, 0, 0, 0 }
  text.src = txt
  Catch[0] = CATCH{ 0, len(txt), 0 }
  Cindex = 1

  if len(txt) == 0 || len(rexp.str) == 0 { return 0 }

  getMods( &rexp, &rexp )

  var loops int
  if (rexp.mods & MOD_ALPHA) > 0 { loops = 1
  } else                         { loops = len(txt) }

  for forward, i := 0, 0; i < loops; i += forward {
    forward  = 1
    Cidx     = 1
    text.pos = 0
    text.init = i
    text.len = len(txt[i:])


    if walker(rexp) {
      if (rexp.mods & MOD_OMEGA) > 0 {
        if text.pos == text.len { return 1
        } else { Cindex = 1 }
      } else if (rexp.mods & MOD_LONLEY   ) > 0 { return 1
      } else if (rexp.mods & MOD_FwrByChar) > 0 || text.pos == 0 { result++
      } else {  forward = text.pos; result++; }
    }
  }

  return result
}

func walker( rexp RE ) bool {
  var track RE
  for oTpos, oCindex, oCidx := text.pos, Cindex, Cidx;
      cutPath( &rexp, &track );
      text.pos, Cindex, Cidx = oTpos, oCindex, oCidx {
    if trekking(&track) { return true }
  }

  return false
}

func trekking( rexp *RE ) bool {
  var track RE
  for tracker( rexp, &track ) {
    if looper( &track ) == false { return false }
  }

  return true
}

func looper(rexp *RE) bool {
  var loops uint32 = 0

  switch rexp.kind {
  case HOOK:
    iCatch := openCatch();
    for loops < rexp.loopsMax && walker( *rexp ) { loops++; }
    if loops >= rexp.loopsMin { closeCatch( iCatch ) }
  case GROUP, PATH:
    for loops < rexp.loopsMax && walker( *rexp ) { loops++; }
  case SET, BACKREF, META, RANGEAB, POINT, SIMPLE:
    for forward := 0; loops < rexp.loopsMax && match( rexp, text.src[text.init + text.pos:], &forward ); {
      text.pos += forward
      loops++;
    }
  }

  if loops < rexp.loopsMin { return false }
  return true
}

func tracker( rexp, track *RE ) bool {
  if len( rexp.str ) == 0 { return false }

  switch rexp.str[0] {
  case ':': cutByLen ( rexp, track, 2,     META    )
  case '.': cutByLen ( rexp, track, 1,     POINT   )
  case '@': cutByLen ( rexp, track, 1 +
          countCharDigits( rexp.str[1:] ), BACKREF )
  case '(': cutPair  ( rexp, track,        GROUP   )
  case '<': cutPair  ( rexp, track,        HOOK    )
  case '[': cutPair  ( rexp, track,        SET     )
  default : cutSimple( rexp, track                 )
  }

  getLoops( rexp, track );
  getMods ( rexp, track );
  return true
}

func cutSimple( rexp, track *RE ) {
  for i, c := range rexp.str {
    switch c {
    case '(', '<', '[', '@', ':', '.':
      cutByLen( rexp, track, i, SIMPLE  ); return
    case '?', '+', '*', '{', '-', '#':
      if( i == 1 ){
        if c == '-' { cutByLen( rexp, track,     3, RANGEAB );
        } else      { cutByLen( rexp, track,     1, SIMPLE  ); }
      } else        { cutByLen( rexp, track, i - 1, SIMPLE  ); }
      return
    }
  }

  cutByLen( rexp, track, len(rexp.str), SIMPLE  );
}

func cutByLen( rexp, track *RE, length int, kind uint8 ){
  *track     = *rexp
  track.str  = rexp.str[:length]
  rexp.str   = rexp.str[length:]
  track.kind = kind;
}

func cutPath( rexp, track *RE ) bool {
  if len(rexp.str) == 0 { return false }

  *track     = *rexp
  track.kind = PATH
  for i := 0; walkMeta( rexp.str[i:], &i ) < len( rexp.str ); i++ {
    switch rexp.str[ i ] {
    case '<': i += walkPair( rexp.str[i:], [2]rune{ '<', '>' } )
    case '(': i += walkPair( rexp.str[i:], [2]rune{ '(', ')' } )
    case '[': i += walkPair( rexp.str[i:], [2]rune{ '[', ']' } )
    case '|':
      track.str = rexp.str[:i]
      rexp.str  = rexp.str[i+1:]
      return true
    }
  }

  rexp.str = ""
  return true
}

func cutPair( rexp, track *RE, kind uint8 ){
  *track       = *rexp;
  track.kind   = kind;

  switch kind {
  case HOOK : track.str = rexp.str[ 1 : walkPair( rexp.str, [2]rune{ '<', '>' } )]
  case GROUP: track.str = rexp.str[ 1 : walkPair( rexp.str, [2]rune{ '(', ')' } )]
  case SET  : track.str = rexp.str[ 1 : walkPair( rexp.str, [2]rune{ '[', ']' } )]
  }

  rexp.str  = rexp.str[len(track.str) + 2:]
}

func walkPair( str string, pair [2]rune ) int {
  deep := 0
  for i, c := range str {
    switch c {
    case pair[0] : deep++
    case pair[1] : deep--
    }

    if deep == 0 { return i }
  }

  return len(str)
}

func walkMeta( str string, n *int ) int {
  for i := 0; i < len( str ); i += 2 {
    if str[i] != ':'  { *n += i; return *n }
  }

  *n += len( str )
  return *n
}

func getMods( rexp, track *RE ){
  if len( rexp.str ) > 0 && rexp.str[ 0 ] == '#' {
    for i, c := range( rexp.str[1:] ) {
      switch c {
      case '^': track.mods |=  MOD_ALPHA
      case '$': track.mods |=  MOD_OMEGA
      case '?': track.mods |=  MOD_LONLEY
      case '~': track.mods |=  MOD_FwrByChar
      case '*': track.mods |=  MOD_COMMUNISM
      case '/': track.mods &=  MOD_CAPITALISM
      default : rexp.str = rexp.str[i+1:]; return
      }
    }

    rexp.str = ""
  }
}

func getLoops( rexp, track *RE ){
  pos := 0;
  track.loopsMin, track.loopsMax = 1, 1

  if len( rexp.str ) > 0 {
    switch rexp.str[0] {
    case '?' : pos = 1; track.loopsMin = 0; track.loopsMax =   1;
    case '+' : pos = 1; track.loopsMin = 1; track.loopsMax = INF;
    case '*' : pos = 1; track.loopsMin = 0; track.loopsMax = INF;
    case '{' : pos = 1
      track.loopsMin = aToi( rexp.str[pos:] )
      pos += countCharDigits( rexp.str[pos:] )

      if rexp.str[pos] == '}' {
              track.loopsMax = track.loopsMin;
              pos += 1
      } else if rexp.str[pos:pos+2] == ",}" {
              pos += 2
              track.loopsMax = INF
      } else if rexp.str[pos] == ',' {
              pos += 1
              track.loopsMax = aToi( rexp.str[pos:] )
              pos += countCharDigits( rexp.str[pos:] ) + 1
      }
    }

    rexp.str = rexp.str[pos:]
  }
}

func match( rexp *RE, txt string, forward *int ) bool {
  switch rexp.kind {
  case POINT  : return matchPoint  (  rexp, txt, forward )
  case SET    : return matchSet    ( *rexp, txt, forward )
  case BACKREF: return matchBackRef(  rexp, txt, forward )
  case RANGEAB: return matchRange  (  rexp, txt, forward )
  case META   : return matchMeta   (  rexp, txt, forward )
  default     : return matchText   (  rexp, txt, forward )
  }
}

func matchPoint( rexp *RE, txt string, forward *int ) bool {
  *forward = 0

  if len(txt) < len(rexp.str) { return false }
  *forward = 1
  return true
}

func matchText( rexp *RE, txt string, forward *int ) bool {
  *forward = len(rexp.str)

  if len(txt) < *forward { return false }


  if (rexp.mods & MOD_COMMUNISM) > 0 {
    return strnEqlCommunist( txt, rexp.str, *forward )
  } else {
    return txt[:*forward] == rexp.str
  }
}

func matchRange( rexp *RE, txt string, forward *int ) bool {
  if len(txt) < 1 { return false }

  *forward = 1
  if (rexp.mods & MOD_COMMUNISM) > 0 {
    chr := toLower( rune(txt[0]) )
    return chr >= toLower( rune(rexp.str[ 0 ]) ) && chr <= toLower( rune(rexp.str[ 2 ]) )
  }

  return txt[0] >= rexp.str[ 0 ] && txt[0] <= rexp.str[ 2 ];
}

func matchMeta( rexp *RE, txt string, forward *int ) bool {
  if len(txt) < 1 { return false }
  *forward = 1

  r := rune(txt[0])
  switch rexp.str[1] {
  case 'a' : return  isAlpha( r )
  case 'A' : return !isAlpha( r )
  case 'd' : return  isDigit( r )
  case 'D' : return !isDigit( r )
  case 'w' : return  isAlnum( r )
  case 'W' : return !isAlnum( r )
  case 's' : return  isSpace( r )
  case 'S' : return !isSpace( r )
  default  : return txt[0] == rexp.str[1]
  }
}

func matchSet( rexp RE, txt string, forward *int ) bool {
  if len(txt) < 1 { return false }

  reverse := rexp.str[0] == '^'
  if reverse { rexp.str = rexp.str[1:] }
  *forward = 1

  var result bool
  var track RE
  for tracker( &rexp, &track ) {
     switch track.kind {
     case GROUP:
             result = walker( track );
     case RANGEAB,  META, POINT:
             result = match( &track, txt, forward )
     default:
       if (track.mods & MOD_COMMUNISM)  > 0 {
         result = findRuneCommunist( track.str, rune(text.src[ text.init + text.pos ] ) )
       } else {
         result = strnchr( track.str, rune( text.src[ text.init + text.pos ]) )
       }
     }

    if result { return !reverse }
  }

  return reverse
}

func matchBackRef( rexp *RE, txt string, forward *int ) bool {
  if len(txt) < 1 { return false }

  backRefId    := aToi( rexp.str[1:] )
  backRefIndex := lastIdCatch( backRefId )
  strCatch     := GetCatch( backRefIndex )
  if strCatch == ""  ||
     text.len - text.init + text.pos < int(len(strCatch)) ||
     strCatch != text.src[text.init + text.pos:text.init + text.pos + len(strCatch)] {
    return false;
  }

  *forward = len(GetCatch( backRefIndex ))
  return true
}

func lastIdCatch( id uint32 ) uint32 {
  for index := Cindex - 1; index > 0; index-- {
    if Catch[ index ].id == id { return index }
  }

  return MAXCATCH;
}

func openCatch() (index uint32) {
  if Cindex < MAXCATCH {
    index = Cindex
    Cindex++
    Catch[index] = CATCH{ text.init + text.pos, text.init + text.pos, Cidx }
    Cidx++
  } else { index = MAXCATCH }

  return index
}

func closeCatch( index uint32 ){
  if index < MAXCATCH {
    Catch[index].end = text.init + text.pos;
  }
}

func TotCatch() uint32 { return Cindex - 1 }

func GetCatch( index uint32 ) string {
  if index > 0 && index < Cindex {
    return text.src[ Catch[index].init : Catch[index].end ]
  }

  return ""
}

func RplCatch( rplStr string, id uint32 ) string {
  var result []byte
  last := 0

  for index := uint32(1); index < Cindex; index++ {
    if Catch[index].id == id {
       for ; last < Catch[index].init; last++ {
          result = append( result, text.src[ last ] )
       }

       for i := 0; i < len( rplStr ); i++ {
          result = append( result, rplStr[ i ] )
       }

       last = Catch[index].end
    }
  }

  for ;last < len(text.src); last++ {
    result = append( result, text.src[ last ] )
  }

  return string(result)
}

func PutCatch( pText string ) string {
  var result []byte

  for i := 0; i < len(pText); {
    if pText[i] == '#' {
      i++
      if len(pText[i:]) > 0 && pText[i] == '#' {
        i++
        result = append( result, '#' );
      } else {
        num := aToi( pText[i:] )
        ary := GetCatch( num )
        for c := 0; c < len(ary); c++ {
          result = append( result, ary[c] )
        }
        i += countCharDigits( pText[i:] )
      }
    } else { result = append( result, pText[i] ); i++ }
  }

  return string(result)
}
