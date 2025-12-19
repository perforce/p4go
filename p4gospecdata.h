/*******************************************************************************

Copyright (c) 2025, Perforce Software, Inc.  All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

1.  Redistributions of source code must retain the above copyright
    notice, this list of conditions and the following disclaimer.

2.  Redistributions in binary form must reproduce the above copyright
    notice, this list of conditions and the following disclaimer in the
    documentation and/or other materials provided with the distribution.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
ARE DISCLAIMED. IN NO EVENT SHALL PERFORCE SOFTWARE, INC. BE LIABLE FOR ANY
DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
(INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

*******************************************************************************/

#ifndef P4GOSPECDATA_H
#define P4GOSPECDATA_H

// Forward declarations - actual includes are in the .cpp file
class SpecData;
class SpecElem;
class StrDict;
class StrBufDict;
class StrPtr;
class StrBuf;
class Error;

class P4GoSpecData : public SpecData
{
public:
    P4GoSpecData( StrDict * d ) : dict(d), owns_dict(false) {}
    P4GoSpecData() : dict(new StrBufDict), owns_dict(true) {}
    virtual ~P4GoSpecData() { if( owns_dict && dict ) delete dict; }

    virtual StrPtr * GetLine( SpecElem *sd, int x, const char **cmt );
    virtual void     SetLine( SpecElem *sd, int x, const StrPtr *val, Error *e );
    virtual void     Comment( SpecElem *sd, int x, const char **wv, int nl, Error *e );
    
    StrDict * GetDict() { return dict; }

private:
    StrDict *   dict;
    StrBuf      last;
    bool        owns_dict;
};

#endif // P4GOSPECDATA_H