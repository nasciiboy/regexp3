package regexp3

const inf = 1073741824 // 2^30

const (
  modAlpha      uint8 = 1
  modOmega      uint8 = 2
  modLonley     uint8 = 4
  modFwrByChar  uint8 = 8
  modCommunism  uint8 = 16
  modNegative   uint8 = 128
  modPositive   uint8 = ^modNegative
  modCapitalism uint8 = ^modCommunism
)

const ( rePath uint8 = iota; reGroup; reHook; reSet; reBackref; reMeta; reRangeab; reUTF8; rePoint; reSimple )

type reStruct struct {
  str                string
  reType             uint8
  mods               uint8
  loopsMin, loopsMax int
}

type catchInfo struct {
  init, end, id int
}

type RE struct {
  txt, re      string
  result       int

  txtInit      int
  txtLen       int
  txtPos       int

  catches      []catchInfo
  catchIndex   int
  catchIdIndex int
}

func (r *RE) Find( txt, re string ) bool {
  if r.Match( txt, re ) > 0 { return true }

  return false
}

func (r *RE) Match( txt, re string ) int {
  rexp, loops := reStruct{ str: re, reType: rePath }, len(txt)
  r.txt, r.re  = txt, re
  r.result     = 0
  r.catches    = make( []catchInfo, 32 )
  r.catchIndex = 1

  if len(txt) == 0 || len(re) == 0 { return 0 }

  getMods( &rexp, &rexp )

  if (rexp.mods & modAlpha) > 0 { loops = 1 }

  for forward, i, ocindex := 0, 0, 0; i < loops; i += forward {
    forward, r.catchIdIndex       = utf8meter( txt[i:] ), 1
    r.txtPos, r.txtInit, r.txtLen = 0, i, len( txt[i:] )
    ocindex                       = r.catchIndex

    if r.walker( rexp ) {
      if (rexp.mods & modOmega) > 0 {
        if r.txtPos == r.txtLen                                 { r.result = 1; return 1
        } else { r.catchIndex = 1 }
      } else if (rexp.mods & modLonley   ) > 0                  { r.result = 1; return 1
      } else if (rexp.mods & modFwrByChar) > 0 || r.txtPos == 0 { r.result++
      } else {   forward = r.txtPos;                              r.result++; }
    } else { r.catchIndex = ocindex }
  }

  return r.result
}

func (r *RE) walker( rexp reStruct ) bool {
  var track reStruct
  for oTextPos, oCatchIndex, oCatchIdIndex := r.txtPos, r.catchIndex, r.catchIdIndex;
      cutByType( &rexp, &track, rePath );
      r.txtPos, r.catchIndex, r.catchIdIndex = oTextPos, oCatchIndex, oCatchIdIndex {
    if r.trekking( &track ) { return true }
  }

  return false
}

func (r *RE) trekking( rexp *reStruct ) bool {
  var track reStruct
  for result := false; tracker( rexp, &track ); {
    switch track.reType {
    case reHook:
      iCatch := r.openCatch();
      result  = r.loopGroup( &track )
      if result { r.closeCatch( iCatch ) }
    case reGroup, rePath:
      result  = r.loopGroup( &track )
    case reSet:
      if track.str[0] == '^' {
        track.str = track.str[1:]
        if (track.mods & modNegative) > 0 { track.mods &=  modPositive
        } else                            { track.mods |=  modNegative }
      }
      fallthrough
    default: result = r.looper( &track ) // case reBackref, reMeta, reRangeab, reUTF8, rePoint, reSimple:
    }

    if result == false { return false }
  }

  return true
}

func (r *RE) looper( rexp *reStruct ) bool {
  loops := 0

  if (rexp.mods & modNegative) > 0 {
    for forward := 0; loops < rexp.loopsMax && r.txtPos < r.txtLen && !r.match( rexp, r.txt[r.txtInit + r.txtPos:], &forward ); {
      r.txtPos += utf8meter( r.txt[r.txtInit + r.txtPos:] )
      loops++;
    }
  } else {
    for forward := 0; loops < rexp.loopsMax && r.txtPos < r.txtLen &&  r.match( rexp, r.txt[r.txtInit + r.txtPos:], &forward ); {
      r.txtPos += forward
      loops++;
    }
  }

  if loops < rexp.loopsMin { return false }
  return true
}

func (r *RE) loopGroup( rexp *reStruct ) bool {
  loops, textxtPos := 0, r.txtPos;

  if (rexp.mods & modNegative) > 0 {
    for loops < rexp.loopsMax && !r.walker( *rexp ) {
      textxtPos++;
      r.txtPos = textxtPos;
      loops++;
    }

    r.txtPos = textxtPos;
  } else {
    for loops < rexp.loopsMax && r.walker( *rexp ) {
      loops++;
    }
  }

  if loops < rexp.loopsMin { return false  }
  return true
}

func tracker( rexp, track *reStruct ) bool {
  if len( rexp.str ) == 0 { return false }

  if rexp.str[0] > 127 {
    cutByLen( rexp, track, utf8meter( rexp.str ), reUTF8 )
  } else {
    switch rexp.str[0] {
    case ':': cutByLen ( rexp, track, 2,     reMeta    )
    case '.': cutByLen ( rexp, track, 1,     rePoint   )
    case '@': cutByLen ( rexp, track, 1 +
            countCharDigits( rexp.str[1:] ), reBackref )
    case '(': cutByType( rexp, track,        reGroup   )
    case '<': cutByType( rexp, track,        reHook    )
    case '[': cutByType( rexp, track,        reSet     )
    default : cutSimple( rexp, track                   )
    }
  }

  getLoops( rexp, track );
  getMods ( rexp, track );
  return true
}

func cutSimple( rexp, track *reStruct ){
  for i, c := range rexp.str {
    if c > 127 {
      cutByLen( rexp, track, i, reSimple  ); return
    } else {
      switch c {
      case '(', '<', '[', '@', ':', '.':
        cutByLen( rexp, track, i, reSimple  ); return
      case '?', '+', '*', '{', '#':
        if i == 1 { cutByLen( rexp, track,     1, reSimple  )
        } else    { cutByLen( rexp, track, i - 1, reSimple  ) }
        return
      }
    }
  }

  cutByLen( rexp, track, len(rexp.str), reSimple  );
}

func cutByLen( rexp, track *reStruct, length int, reType uint8 ){
  *track       = *rexp
  track.str    = rexp.str[:length]
  rexp.str     = rexp.str[length:]
  track.reType = reType;
}

func cutByType( rexp, track *reStruct, reType uint8 ) bool {
  if len(rexp.str) == 0 { return false }

  *track       = *rexp
  track.reType = reType
  for i, deep, cut := 0, 0, false; walkMeta( rexp.str[i:], &i ) < len( rexp.str ); i++ {
    switch rexp.str[ i ] {
    case '(', '<': deep++
    case ')', '>': deep--
    case '[': i += walkSet( rexp.str[i:] )
    }

    switch reType {
    case reHook, reGroup: cut = deep == 0
    case reSet          : cut = rexp.str[ i ] == ']'
    case rePath         : cut = rexp.str[ i ] == '|' && deep == 0
    }

    if cut {
      track.str  = track.str[:i]
      rexp.str   = rexp.str[i + 1:]
      if reType != rePath { track.str = track.str[1:] }
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
    if str[i] != ':' { *n += i; return *n }
  }

  *n += len( str )
  return *n
}

func getMods( rexp, track *reStruct ){
  track.mods &= modPositive

  if len( rexp.str ) > 0 && rexp.str[ 0 ] == '#' {
    for i, c := range rexp.str[1:] {
      switch c {
      case '^': track.mods |= modAlpha
      case '$': track.mods |= modOmega
      case '?': track.mods |= modLonley
      case '~': track.mods |= modFwrByChar
      case '*': track.mods |= modCommunism
      case '/': track.mods &= modCapitalism
      case '!': track.mods |= modNegative
      default : rexp.str    = rexp.str[i+1:]; return
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
    case '+' : pos = 1; track.loopsMin = 1; track.loopsMax = inf;
    case '*' : pos = 1; track.loopsMin = 0; track.loopsMax = inf;
    case '{' : pos = 1
      track.loopsMin = aToi( rexp.str[pos:] )
      pos += countCharDigits( rexp.str[pos:] )

      if rexp.str[pos] == '}' {
        track.loopsMax = track.loopsMin;
        pos += 1
      } else if rexp.str[pos:pos+2] == ",}" {
        pos += 2
        track.loopsMax = inf
      } else if rexp.str[pos] == ',' {
        pos += 1
        track.loopsMax = aToi( rexp.str[pos:] )
        pos += countCharDigits( rexp.str[pos:] ) + 1
      }
    }

    rexp.str = rexp.str[pos:]
  }
}

func (r *RE) match( rexp *reStruct, txt string, forward *int ) bool {
  switch rexp.reType {
  case rePoint  : *forward = utf8meter( txt );  return true
  case reSet    : return r.matchSet    ( *rexp, txt, forward )
  case reBackref: return r.matchBackRef(  rexp, txt, forward )
  case reRangeab: return matchRange    (  rexp, txt, forward )
  case reMeta   : return matchMeta     (  rexp, txt, forward )
  default       : return matchText     (  rexp, txt, forward )
  }
}

func matchText( rexp *reStruct, txt string, forward *int ) bool {
  *forward = len(rexp.str)

  if len(txt) < *forward { return false }

  if (rexp.mods & modCommunism) > 0 {
    return strnEqlCommunist( txt, rexp.str, *forward )
  }

  return txt[:*forward] == rexp.str
}

func matchRange( rexp *reStruct, txt string, forward *int ) bool {
  *forward = 1
  if (rexp.mods & modCommunism) > 0 {
    chr := toLower( rune(txt[0]) )
    return chr >= toLower( rune(rexp.str[ 0 ]) ) && chr <= toLower( rune(rexp.str[ 2 ]) )
  }

  return txt[0] >= rexp.str[ 0 ] && txt[0] <= rexp.str[ 2 ];
}

func matchMeta( rexp *reStruct, txt string, forward *int ) bool {
  var f func( r rune ) bool
  *forward = 1

  switch rexp.str[1] {
  case 'a' : return isAlpha( rune(txt[0]) )
  case 'A' : f = isAlpha
  case 'd' : return isDigit( rune(txt[0]) )
  case 'D' : f = isDigit
  case 'w' : return isAlnum( rune(txt[0]) )
  case 'W' : f = isAlnum
  case 's' : return isSpace( rune(txt[0]) )
  case 'S' : f = isSpace
  case 'b' : return isBlank( rune(txt[0]) )
  case 'B' : f = isBlank
  case '&' : if txt[0] < 128 { return false }
    *forward = utf8meter( txt )
    return true
  default  : return txt[0] == rexp.str[1]
  }

  if f( rune(txt[0]) ) { return false }
  *forward = utf8meter( txt )
  return true
}

func (r *RE) matchSet( rexp reStruct, txt string, forward *int ) bool {
  *forward = 1

  var result bool
  var track reStruct
  for trackerSet( &rexp, &track ) {
    switch track.reType {
    case reRangeab, reUTF8, reMeta:
      result = r.match( &track, txt, forward )
    default:
      if (track.mods & modCommunism)  > 0 {
        result = findRuneCommunist( track.str, rune( txt[ 0 ] ) )
      } else {
        result = strnchr( track.str, rune( txt[ 0 ] ) )
      }
    }

    if result { return true }
  }

  return false
}

func trackerSet( rexp, track *reStruct ) bool {
  if len( rexp.str ) == 0 { return false }

  if rexp.str[0] > 127 {
    cutByLen( rexp, track, utf8meter( rexp.str ), reUTF8 )
  } else if rexp.str[0] == ':' {
    cutByLen ( rexp, track, 2, reMeta  )
  } else {
    for i := 0; i < len( rexp.str ); i++ {
      if rexp.str[i] > 127 {
        cutByLen( rexp, track, i, reSimple  ); goto setLM;
      } else {
        switch rexp.str[i] {
        case ':': cutByLen( rexp, track, i, reSimple  ); goto setLM;
        case '-':
          if i == 1 { cutByLen( rexp, track,     3, reRangeab )
          } else    { cutByLen( rexp, track, i - 1, reSimple  ) }

          goto setLM;
        }
      }
    }

    cutByLen( rexp, track, len( rexp.str ), reSimple  );
  }

 setLM:
  track.loopsMin, track.loopsMax = 1, 1
  track.mods &= modPositive
  return true
}

func (r *RE) matchBackRef( rexp *reStruct, txt string, forward *int ) bool {
  backRefId    := aToi( rexp.str[1:] )
  backRefIndex := r.lastIdCatch( backRefId )
  strCatch     := r.GetCatch( backRefIndex )
  *forward      = len(strCatch)

  if strCatch == "" || len( txt ) < *forward || strCatch != txt[:*forward] { return false }

  return true
}

func (r *RE) lastIdCatch( id int ) int {
  for index := r.catchIndex - 1; index > 0; index-- {
    if r.catches[ index ].id == id { return index }
  }

  return len(r.catches);
}

func (r *RE) openCatch() (index int) {
  index = r.catchIndex

  if r.catchIndex < len(r.catches) {
    r.catches[index] = catchInfo{ r.txtInit + r.txtPos, r.txtInit + r.txtPos, r.catchIdIndex }
  } else {
    r.catches = append( r.catches, catchInfo{ r.txtInit + r.txtPos, r.txtInit + r.txtPos, r.catchIdIndex } )
  }

  r.catchIndex++
  r.catchIdIndex++
  return
}

func (r *RE) closeCatch( index int ){
  if index < r.catchIndex {
    r.catches[index].end = r.txtInit + r.txtPos
  }
}

func (r *RE) Result  () int { return r.result }

func (r *RE) TotCatch() int { return r.catchIndex - 1 }

func (r *RE) GetCatch( index int ) string {
  if index < 1 || index >= r.catchIndex { return "" }
  return r.txt[ r.catches[index].init : r.catches[index].end ]
}

func (r *RE) GpsCatch( index int ) int {
  if index < 1 || index >= r.catchIndex { return 0 }
  return r.catches[index].init
}

func (r *RE) LenCatch( index int ) int {
  if index < 1 || index >= r.catchIndex { return 0 }
  return r.catches[index].end - r.catches[index].init
}

func (r *RE) RplCatch( rplStr string, id int ) (result string) {
  last := 0

  for index := 1; index < r.catchIndex; index++ {
    if r.catches[index].id == id {
      result += r.txt[last:r.catches[index].init]
      result += rplStr
      last    = r.catches[index].end
    }
  }

  if last < len(r.txt) { result += r.txt[last:] }

  return string(result)
}

func (r *RE) PutCatch( pStr string ) (result string) {
  for i := 0; i < len(pStr); {
    if pStr[i] == '#' {
      i++
      if len(pStr[i:]) > 0 && pStr[i] == '#' {
        i++
        result += "#"
      } else {
        result += r.GetCatch( aToi( pStr[i:] ) )
        i      += countCharDigits ( pStr[i:] )
      }
    } else { result += pStr[i:i+1]; i++ }
  }

  return
}
