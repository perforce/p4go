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

#include <p4/clientapi.h>
#include <p4/spec.h>
#include <p4/strtable.h>
#include "p4gospecdata.h"
#include <iostream>

using namespace std;

StrPtr *
P4GoSpecData::GetLine( SpecElem *sd, int x, const char **cmt )
{
    StrPtr *val = dict->GetVar( sd->tag );
    if( !val ) return 0;

    if( !sd->IsList() )
    {
        last = *val;
        return &last;
    }

    // It's a list field - check if we have array values
    // Values should be stored as "fieldname[0]", "fieldname[1]", etc.
    StrBuf arrayKey;
    arrayKey << sd->tag << "[" << x << "]";
    
    val = dict->GetVar( arrayKey );
    if( !val ) return 0;

    last = *val;
    return &last;
}

void    
P4GoSpecData::SetLine( SpecElem *sd, int x, const StrPtr *v, Error *e )
{
    if( sd->IsList() )
    {
        // For list fields, store with array index
        StrBuf arrayKey;
        arrayKey << sd->tag << "[" << x << "]";
        dict->SetVar( arrayKey.Text(), *v );
    }
    else
    {
        // For scalar fields, store directly
        dict->SetVar( sd->tag, *v );
    }
}

void
P4GoSpecData::Comment( SpecElem *sd, int x, const char **wv, int nl, Error *e )
{
    // Ignore comments (similar to PythonSpecData for non-list fields)
    if( !sd->IsList() )
        return;

    const char * comment = *wv;
    
    if( nl )
    {
        // New line comment - store as new array element
        StrBuf arrayKey;
        arrayKey << sd->tag << "[" << x << "]";
        dict->SetVar( arrayKey.Text(), comment );
    }
    else
    {
        // Append to previous comment
        StrBuf arrayKey;
        arrayKey << sd->tag << "[" << (x-1) << "]";
        StrPtr *prev = dict->GetVar( arrayKey );
        
        if( prev )
        {
            StrBuf newContent;
            newContent << prev->Text() << " " << comment;
            dict->SetVar( arrayKey.Text(), newContent );
        }
    }
}