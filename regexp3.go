package regexp3

const INF = 1073741824 // 2^30

const MOD_ALPHA      = 1
const MOD_OMEGA      = 2
const MOD_LONLEY     = 4
const MOD_FwrByChar  = 8
const MOD_COMMUNISM  = 16
const MOD_CAPITALISM = 0xEF
const MOD_NEGATIVE   = 128
const MOD_POSITIVE   = 0x7F

type TEXT struct {
  src string
  init int
  len  int
  pos  int
}

var pText *TEXT

type CATCH struct {
  init, end int
  id  uint32
}

const MAXCATCH = 16
var   pCatch *[MAXCATCH]CATCH
var   Cidx uint32
var   pCindex *uint32

const ( PATH = iota; GROUP; HOOK; SET; BACKREF; META; RANGEAB; POINT; SIMPLE )

type reStruct struct {
  str                string
  kind               uint8
  mods               uint8
  loopsMin, loopsMax uint32
}

type RE struct {
  Txt, Re string
  result uint
  text TEXT
  catch[MAXCATCH] CATCH
  catchIndex uint32
}

func (p *RE) Match( txt, re string ) (result uint) {
  p.Txt, p.Re  = txt, re
  rexp        := reStruct{ p.Re, PATH, 0, 0, 0 }
  pText        = &p.text
  p.text.src   = txt
  pCatch       = &p.catch
  p.catchIndex = 1
  pCindex      = &p.catchIndex

  if len(txt) == 0 || len(rexp.str) == 0 { return 0 }

  getMods( &rexp, &rexp )

  var loops int
  if (rexp.mods & MOD_ALPHA) > 0 { loops = 1
  } else                         { loops = len(txt) }

  for forward, i := 0, 0; i < loops; i += forward {
    forward  = 1
    Cidx     = 1
    p.text.pos  = 0
    p.text.init = i
    p.text.len  = len(txt[i:])


    if walker(rexp) {
      if (rexp.mods & MOD_OMEGA) > 0 {
        if p.text.pos == p.text.len {                                return 1
        } else { p.catchIndex = 1 }
      } else if (rexp.mods & MOD_LONLEY   ) > 0 {                    return 1
      } else if (rexp.mods & MOD_FwrByChar) > 0 || p.text.pos == 0 { result++
      } else {   forward = p.text.pos;                               result++; }
    }
  }

  return result
}

func walker( rexp reStruct ) bool {
  var track reStruct
  for oTpos, oCindex, oCidx := pText.pos, *pCindex, Cidx;
      cutPath( &rexp, &track );
      pText.pos, *pCindex, Cidx = oTpos, oCindex, oCidx {
    if trekking(&track) { return true }
  }

  return false
}

func trekking( rexp *reStruct ) bool {
  var track reStruct
  for result := false; tracker( rexp, &track ); {
    switch track.kind {
    case HOOK:
      iCatch := openCatch();
      result  = loopGroup( &track )
      if result { closeCatch( iCatch ) }
    case GROUP, PATH:
      result  = loopGroup( &track )
    case SET:
      if track.str[0] == '^' {
        track.str = track.str[1:]
        if (track.mods & MOD_NEGATIVE) > 0 { track.mods &=  MOD_POSITIVE
        } else                             { track.mods |=  MOD_NEGATIVE }
      }
      fallthrough
    case BACKREF, META, RANGEAB, POINT, SIMPLE:
      result = looper( &track )
    }

    if result == false { return false }
  }

  return true
}

func looper( rexp *reStruct ) bool {
  var loops uint32 = 0

  if (rexp.mods & MOD_NEGATIVE) > 0 {
    for forward := 0; loops < rexp.loopsMax && (pText.pos < pText.len) && !match( rexp, pText.src[pText.init + pText.pos:], &forward ); {
      pText.pos += 1;
      loops++;
    }
  } else {
    for forward := 0; loops < rexp.loopsMax && (pText.pos < pText.len) && match( rexp, pText.src[pText.init + pText.pos:], &forward ); {
      pText.pos += forward
      loops++;
    }
  }

  if loops < rexp.loopsMin { return false }
  return true
}

func loopGroup( rexp *reStruct ) bool {
  loops, textPos := uint32(0), pText.pos;

  if (rexp.mods & MOD_NEGATIVE) > 0 {
    for loops < rexp.loopsMax && !walker( *rexp ) {
      textPos++;
      pText.pos = textPos;
      loops++;
    }

    pText.pos = textPos;
  } else {
    for loops < rexp.loopsMax && walker( *rexp ) {
      loops++;
    }
  }

  if loops < rexp.loopsMin { return false  }
  return true
}


func tracker( rexp, track *reStruct ) bool {
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

func cutSimple( rexp, track *reStruct ) {
  for i, c := range rexp.str {
    switch c {
    case '(', '<', '[', '@', ':', '.':
      cutByLen( rexp, track, i, SIMPLE  ); return
    case '?', '+', '*', '{', '#':
      if i == 1 { cutByLen( rexp, track,     1, SIMPLE  )
      } else    { cutByLen( rexp, track, i - 1, SIMPLE  ) }
      return
    }
  }

  cutByLen( rexp, track, len(rexp.str), SIMPLE  );
}

func cutByLen( rexp, track *reStruct, length int, kind uint8 ){
  *track     = *rexp
  track.str  = rexp.str[:length]
  rexp.str   = rexp.str[length:]
  track.kind = kind;
}

func cutPath( rexp, track *reStruct ) bool {
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

func cutPair( rexp, track *reStruct, kind uint8 ){
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

func getMods( rexp, track *reStruct ){
  track.mods &= MOD_POSITIVE

  if len( rexp.str ) > 0 && rexp.str[ 0 ] == '#' {
    for i, c := range( rexp.str[1:] ) {
      switch c {
      case '^': track.mods |=  MOD_ALPHA
      case '$': track.mods |=  MOD_OMEGA
      case '?': track.mods |=  MOD_LONLEY
      case '~': track.mods |=  MOD_FwrByChar
      case '*': track.mods |=  MOD_COMMUNISM
      case '/': track.mods &=  MOD_CAPITALISM
      case '!': track.mods |=  MOD_NEGATIVE
      default : rexp.str = rexp.str[i+1:]; return
      }
    }

    rexp.str = ""
  }
}

func getLoops( rexp, track *reStruct ){
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

func match( rexp *reStruct, txt string, forward *int ) bool {
  switch rexp.kind {
  case POINT  : return matchPoint  (  rexp, txt, forward )
  case SET    : return matchSet    ( *rexp, txt, forward )
  case BACKREF: return matchBackRef(  rexp, txt, forward )
  case RANGEAB: return matchRange  (  rexp, txt, forward )
  case META   : return matchMeta   (  rexp, txt, forward )
  default     : return matchText   (  rexp, txt, forward )
  }
}

func matchPoint( rexp *reStruct, txt string, forward *int ) bool {
  *forward = 0

  if len(txt) < len(rexp.str) { return false }
  *forward = 1
  return true
}

func matchText( rexp *reStruct, txt string, forward *int ) bool {
  *forward = len(rexp.str)

  if len(txt) < *forward { return false }


  if (rexp.mods & MOD_COMMUNISM) > 0 {
    return strnEqlCommunist( txt, rexp.str, *forward )
  } else {
    return txt[:*forward] == rexp.str
  }
}

func matchRange( rexp *reStruct, txt string, forward *int ) bool {
  if len(txt) < 1 { return false }

  *forward = 1
  if (rexp.mods & MOD_COMMUNISM) > 0 {
    chr := toLower( rune(txt[0]) )
    return chr >= toLower( rune(rexp.str[ 0 ]) ) && chr <= toLower( rune(rexp.str[ 2 ]) )
  }

  return txt[0] >= rexp.str[ 0 ] && txt[0] <= rexp.str[ 2 ];
}

func matchMeta( rexp *reStruct, txt string, forward *int ) bool {
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

func matchSet( rexp reStruct, txt string, forward *int ) bool {
  if len(txt) < 1 { return false }

  *forward = 1

  var result bool
  var track reStruct
  for trackerSet( &rexp, &track ) {
    switch track.kind {
    case RANGEAB,  META:
      result = match( &track, txt, forward )
    default:
      if (track.mods & MOD_COMMUNISM)  > 0 {
        result = findRuneCommunist( track.str, rune(pText.src[ pText.init + pText.pos ] ) )
      } else {
        result = strnchr( track.str, rune( pText.src[ pText.init + pText.pos ]) )
      }
    }

    if result { return true }
  }

  return false
}

func trackerSet( rexp, track *reStruct ) bool {
  if len( rexp.str ) == 0 { return false }

  if rexp.str[0] == ':' { cutByLen ( rexp, track, 2, META  )
  } else {
    for i := 0; i < len( rexp.str ); i++ {
      switch rexp.str[i] {
      case ':': cutByLen( rexp, track, i, SIMPLE  ); goto setL;
      case '-':
        if i == 1 { cutByLen( rexp, track,     3, RANGEAB )
        } else    { cutByLen( rexp, track, i - 1, SIMPLE  ) }

        goto setL;
      }
    }

    cutByLen( rexp, track, len( rexp.str ), SIMPLE  );
  }

 setL:
  track.loopsMin, track.loopsMax = 1, 1
  return true
}

func matchBackRef( rexp *reStruct, txt string, forward *int ) bool {
  if len(txt) < 1 { return false }

  backRefId    := aToi( rexp.str[1:] )
  backRefIndex := lastIdCatch( backRefId )
  strCatch     := getCatch( backRefIndex )
  if strCatch == ""  ||
    len( txt ) < int(len(strCatch)) ||
    strCatch != txt[:len(strCatch)] {
    return false;
  }

  *forward = len(getCatch( backRefIndex ))
  return true
}

func lastIdCatch( id uint32 ) uint32 {
  for index := *pCindex - 1; index > 0; index-- {
    if pCatch[ index ].id == id { return index }
  }

  return MAXCATCH;
}

func openCatch() (index uint32) {
  if *pCindex < MAXCATCH {
    index = *pCindex
    *pCindex++
    pCatch[index] = CATCH{ pText.init + pText.pos, pText.init + pText.pos, Cidx }
    Cidx++
  } else { index = MAXCATCH }

  return index
}

func closeCatch( index uint32 ){
  if index < MAXCATCH {
    pCatch[index].end = pText.init + pText.pos;
  }
}

func getCatch( index uint32 ) string {
  if index > 0 && index < *pCindex {
    return pText.src[ pCatch[index].init : pCatch[index].end ]
  }

  return ""
}

func (p *RE) TotCatch() uint32 { return p.catchIndex - 1 }

func (p *RE) GetCatch( index uint32 ) string {
  if index > 0 && index < p.catchIndex {
    return p.Txt[ p.catch[index].init : p.catch[index].end ]
  }

  return ""
}

func (p *RE) RplCatch( rplStr string, id uint32 ) string {
  var result []byte
  last := 0

  for index := uint32(1); index < p.catchIndex; index++ {
    if p.catch[index].id == id {
       for ; last < p.catch[index].init; last++ {
          result = append( result, p.Txt[ last ] )
       }

       for i := 0; i < len( rplStr ); i++ {
          result = append( result, rplStr[ i ] )
       }

       last = p.catch[index].end
    }
  }

  for ;last < len(p.Txt); last++ {
    result = append( result, p.Txt[ last ] )
  }

  return string(result)
}

func (p *RE) PutCatch( pStr string ) string {
  var result []byte

  for i := 0; i < len(pStr); {
    if pStr[i] == '#' {
      i++
      if len(pStr[i:]) > 0 && pStr[i] == '#' {
        i++
        result = append( result, '#' );
      } else {
        num := aToi( pStr[i:] )
        ary := p.GetCatch( num )
        for c := 0; c < len(ary); c++ {
          result = append( result, ary[c] )
        }
        i += countCharDigits( pStr[i:] )
      }
    } else { result = append( result, pStr[i] ); i++ }
  }

  return string(result)
}
