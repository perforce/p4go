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

#include <p4/clientapi.h>
#include <p4/vararray.h>
#include <p4/strarray.h>
#include <p4/spec.h>
#include <p4/strtable.h>
#include "p4gospecmgr.h"
#include "p4gospecdata.h"
#include "p4goresult.h"

P4GoResults::P4GoResults()
{
    apiLevel = atoi( P4Tag::l_client );
    Reset();
}

P4GoResults::~P4GoResults()
{
    Reset();
}

void
P4GoResults::Reset()
{
    // Remove from the end to prevent moving
    for( int i = Count() - 1; i >= 0; i-- ) {
        Destroy( Get( i ) );
        Remove( i );
    }

    infoCount = 0;
    warnCount = 0;
    errorCount = 0;
    trackCount = 0;
    dictCount = 0;
    specCount = 0;
    stringCount = 0;
}

int
P4GoResults::Compare( const void* r1, const void* r2 ) const
{
    return 0; // don't sort?
}

void
P4GoResults::Destroy( void* r ) const
{
    // Delete the object if it belongs to us
    P4GoResult* res = (P4GoResult*)r;
    if( !res->taken ) {
        switch( res->type ) {
        case STRING:
        case BINARY:
        case TRACK:
            delete res->str;
            break;
        case DICT:
            delete res->dict;
            break;
        case ERROR:
            delete res->err;
            break;
        case SPEC:
            delete res->spec;
            break;
        }
        delete res;
    } else
        res->taken = 2;
}

//
// Direct output - not via a message of any kind. For example,
// binary output.
//
void
P4GoResults::AddOutput( Error* e )
{
    if( e->GetSeverity() == E_EMPTY )
        return;

    P4GoResult* r = new P4GoResult;
    r->type = ERROR;
    r->taken = 0;
    r->err = new Error;
    *r->err = *e;
    Put( r );

    if( e->GetSeverity() == E_INFO )
        infoCount++;
    else if( e->GetSeverity() == E_WARN )
        warnCount++;
    else
        errorCount++;
}

void
P4GoResults::AddOutput( StrPtr o, bool binary )
{
    P4GoResult* r = new P4GoResult;
    r->type = binary ? BINARY : STRING;
    r->taken = 0;
    r->str = new StrBuf;
    *r->str = o;
    Put( r );
    stringCount++;
}

void
P4GoResults::AddOutput( StrDict* d )
{
    P4GoResult* r = new P4GoResult;
    r->type = DICT;
    r->taken = 0;
    r->dict = d;
    Put( r );
    dictCount++;
}

void
P4GoResults::AddOutput( P4GoSpecData* s )
{
    P4GoResult* r = new P4GoResult;
    r->type = SPEC;
    r->taken = 0;
    r->spec = s;
    Put( r );
    specCount++;
}

void
P4GoResults::AddTrack( const char* t )
{
    P4GoResult* r = new P4GoResult;
    r->type = TRACK;
    r->taken = 0;
    r->str = new StrBuf;
    *r->str = t;
    Put( r );
    trackCount++;
}

void
P4GoResults::AddTrack( StrPtr t )
{
    P4GoResult* r = new P4GoResult;
    r->type = TRACK;
    r->taken = 0;
    r->str = new StrBuf;
    *r->str = t;
    Put( r );
    trackCount++;
}

void
P4GoResults::DeleteTrack()
{
    // Rollback any track lines
    for( int i = Count() - 1; i >= 0; i-- ) {
        P4GoResult* res = (P4GoResult*)Get( i );
        if( res->type != TRACK )
            return;
        Destroy( Get( i ) );
        Remove( i );
    }
}

int
P4GoResults::ErrorCount()
{
    return errorCount;
}

int
P4GoResults::WarningCount()
{
    return warnCount;
}

void
P4GoResults::FmtErrors( StrBuf& buf )
{
    // Fmt( "[Error]: ", errors, buf );
}

void
P4GoResults::FmtWarnings( StrBuf& buf )
{
    // Fmt( "[Warning]: ", warnings, buf );
}

void
P4GoResults::Fmt( const char* label, void* ary, StrBuf& buf )
{
    // ID		idJoin;
    // VALUE	s1;
    StrBuf csep;
    // VALUE	rsep;

    buf.Clear();
    // If the array is empty, then we just return
    if( !Count() )
        return;

    // This is the string we'll use to prefix each entry in the array
    csep << "\n\t" << label;

    buf << csep;

    return;
}

char*
P4GoResults::FmtMessage( Error* e )
{
    StrBuf t;
    e->Fmt( t, EF_PLAIN );
    char* s = (char*)malloc( t.Length() + 1 );
    memcpy( s, t.Text(), t.Length() + 1 );
    return s;
}
