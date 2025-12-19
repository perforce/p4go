/*******************************************************************************

Copyright (c) 2024, Perforce Software, Inc.  All rights reserved.

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

#ifndef P4GOSPECMGR_H
#define P4GOSPECMGR_H

// Forward declarations
class StrBufDict;
class StrDict;
class StrPtr;
class StrBuf;
class Error;
class P4GoSpecData;

class P4GoSpecMgr
{
  public:
    P4GoSpecMgr();
    ~P4GoSpecMgr();

    void SetDebug( int i ) { debug = i; }

    void SetArrayConversion( int a ) { convertArray = a; }

    // Clear the spec cache and revert to internal defaults
    void Reset();

    // Add a spec to the cache
    void AddSpecDef( const char* type, StrPtr& specDef );
    void AddSpecDef( const char* type, const char* specDef );

    // Check that a type of spec is known.
    int HaveSpecDef( const char* type );

    //
    // Parse routine: converts strings into Go spec
    //
    P4GoSpecData* StringToSpec( const char* type, const char* spec, Error* e );

    //
    // Format routine. updates a StrBuf object with the form;
    // that can then be converted to a Go string where required.
    //
    void SpecToString( const char* type, P4GoSpecData* spec, StrBuf& b, Error* e );

    //
    // Convert a Perforce StrDict into a P4::Spec object. This is for
    // 2005.2 and later servers where the forms are supplied pre-parsed
    // into a dictionary - we just need to convert them. The specDef
    // argument tells us what type of spec we're converting.
    //
    P4GoSpecData* StrDictToSpec( StrDict* dict, StrPtr* specDef );

    //
    // Return a list of the fields in a given type of spec. Return null
    // if the spec type is not known.
    //
    StrDict* SpecFields( const char* type );

  private:
    void SplitKey( const StrPtr* key, StrBuf& base, StrBuf& index );
    void* NewSpec( StrPtr* specDef );
    StrDict* SpecFields( StrPtr* specDef );

  private:
    int debug;
    int convertArray;
    StrBufDict* specs;
};

#endif // P4GOSPECMGR_H