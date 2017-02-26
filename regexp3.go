package regexp3

const INF = 1073741824 // 2^30

const MOD_ALPHA      = 1
const MOD_OMEGA      = 2
const MOD_LONLEY     = 4
const MOD_FwrByChar  = 8
const MOD_COMMUNISM  = 16
const MOD_CAPITALISM = 0xEF // ~16
const MOD_NEGATIVE   = 128
const MOD_POSITIVE   = 0x7F // ~128

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

var Catch  []CATCH
var Cidx   uint32
var Cindex uint32

const ( PATH = iota; GROUP; HOOK; SET; BACKREF; META; RANGEAB; POINT; SIMPLE )

type reStruct struct {
  str                string
  kind               uint8
  mods               uint8
  loopsMin, loopsMax uint32
}

type RE struct {
  Txt, Re string
  Result  uint
  catch   []CATCH
}

func (p *RE) Match( txt, re string ) (result uint) {
  p.Txt, p.Re  = txt, re
  rexp        := reStruct{ p.Re, PATH, 0, 0, 0 }
  text         = TEXT{ txt, 0, len(txt), 0 }
  Catch        = []CATCH{}
  Catch        = append( Catch, CATCH{ 0, len(txt), 0 } )
  Cindex       = 1

  if len(txt) == 0 || len(rexp.str) == 0 { return 0 }

  getMods( &rexp, &rexp )

  var loops int
  if (rexp.mods & MOD_ALPHA) > 0 { loops = 1
  } else                         { loops = len(txt) }

  for forward, i := 0, 0; i < loops; i += forward {
    forward  = 1
    Cidx     = 1
    text.pos  = 0
    text.init = i
    text.len  = len(txt[i:])

    if walker(rexp) {
      if (rexp.mods & MOD_OMEGA) > 0 {
        if text.pos == text.len                                  { return setData( p, 1 )
        } else { Cindex = 1 }
      } else if (rexp.mods & MOD_LONLEY   ) > 0                  { return setData( p, 1 )
      } else if (rexp.mods & MOD_FwrByChar) > 0 || text.pos == 0 { result++
      } else {   forward = text.pos;                               result++; }
    }
  }

  return setData( p, result )
}

func setData( p *RE, i uint ) uint {
  p.Result = i
  p.catch  = Catch[:Cindex]

  return i
}

func walker( rexp reStruct ) bool {
  var track reStruct
  for oTpos, oCindex, oCidx := text.pos, Cindex, Cidx;
      cutByType( &rexp, &track, PATH );
      text.pos, Cindex, Cidx = oTpos, oCindex, oCidx {
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
    for forward := 0; loops < rexp.loopsMax && (text.pos < text.len) && !match( rexp, text.src[text.init + text.pos:], &forward ); {
      text.pos += 1;
      loops++;
    }
  } else {
    for forward := 0; loops < rexp.loopsMax && (text.pos < text.len) && match( rexp, text.src[text.init + text.pos:], &forward ); {
      text.pos += forward
      loops++;
    }
  }

  if loops < rexp.loopsMin { return false }
  return true
}

func loopGroup( rexp *reStruct ) bool {
  loops, textPos := uint32(0), text.pos;

  if (rexp.mods & MOD_NEGATIVE) > 0 {
    for loops < rexp.loopsMax && !walker( *rexp ) {
      textPos++;
      text.pos = textPos;
      loops++;
    }

    text.pos = textPos;
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
  case '(': cutByType( rexp, track,        GROUP   )
  case '<': cutByType( rexp, track,        HOOK    )
  case '[': cutByType( rexp, track,        SET     )
  default : cutSimple( rexp, track                 )
  }

  getLoops( rexp, track );
  getMods ( rexp, track );
  return true
}

func cutSimple( rexp, track *reStruct ){
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

func cutByType( rexp, track *reStruct, kind uint8 ) bool {
  if len(rexp.str) == 0 { return false }

  *track     = *rexp
  track.kind = kind
  for i, deep, cut := 0, 0, false; walkMeta( rexp.str[i:], &i ) < len( rexp.str ); i++ {
    switch rexp.str[ i ] {
    case '(', '<': deep++
    case ')', '>': deep--
    case '[': i += walkSet( rexp.str[i:] )
    }

    switch kind {
    case HOOK, GROUP: cut = deep == 0
    case SET        : cut = rexp.str[ i ] == ']'
    case PATH       : cut = rexp.str[ i ] == '|' && deep == 0
    }

    if cut {
      track.str = track.str[:i]
      rexp.str  = rexp.str[i + 1:]
      if kind != PATH { track.str = track.str[1:] }
      return true
    }
  }

  rexp.str = ""
  return true
}

func walkSet( str string ) int {
  for i := 0; walkMeta( str[i:], &i ) < len( str ); i++ {
    if str[i] == ']' { return i }
  }

  return len(str);
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
  case 'b' : return  isBlank( r )
  case 'B' : return !isBlank( r )
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
        result = findRuneCommunist( track.str, rune(text.src[ text.init + text.pos ] ) )
      } else {
        result = strnchr( track.str, rune( text.src[ text.init + text.pos ]) )
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
  for index := Cindex - 1; index > 0; index-- {
    if Catch[ index ].id == id { return index }
  }

  return uint32(len(Catch));
}

func openCatch() (index uint32) {
  index = Cindex

  if int(Cindex) < len(Catch) {
    Catch[index] = CATCH{ text.init + text.pos, text.init + text.pos, Cidx }
  } else {
    Catch = append( Catch, CATCH{ text.init + text.pos, text.init + text.pos, Cidx } )
  }

  Cindex++
  Cidx++

  return index
}

func closeCatch( index uint32 ){
  if index < Cindex {
    Catch[index].end = text.init + text.pos;
  }
}

func getCatch( index uint32 ) string {
  if index > 0 && index < Cindex {
    return text.src[ Catch[index].init : Catch[index].end ]
  }

  return ""
}

func (p RE) TotCatch() uint32 { return uint32(len(p.catch)) - 1 }

func (p RE) GetCatch( index uint32 ) string {
  if index > 0 && int(index) < len(p.catch) {
    return p.Txt[ p.catch[index].init : p.catch[index].end ]
  }

  return ""
}

func (p RE) GpsCatch( index uint32 ) int {
  if index > 0 && int(index) < len(p.catch) {
    return p.catch[index].init
  }

  return 0
}

func (p RE) LenCatch( index uint32 ) int {
  if index > 0 && int(index) < len(p.catch) {
    return p.catch[index].end - p.catch[index].init
  }

  return 0
}

func (p RE) RplCatch( rplStr string, id uint32 ) (result string) {
  last := 0

  for index := 1; index < len(p.catch); index++ {
    if p.catch[index].id == id {
      result += p.Txt[last:p.catch[index].init]
      result += rplStr
      last    = p.catch[index].end
    }
  }

  if last < len(p.Txt) { result += p.Txt[last:] }

  return string(result)
}

func (p RE) PutCatch( pStr string ) (result string) {
  for i := 0; i < len(pStr); {
    if pStr[i] == '#' {
      i++
      if len(pStr[i:]) > 0 && pStr[i] == '#' {
        i++
        result += "#"
      } else {
        result += p.GetCatch( aToi( pStr[i:] ) )
        i      += countCharDigits ( pStr[i:] )
      }
    } else { result += pStr[i:i+1]; i++ }
  }

  return
}
