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
#include <p4/strtable.h>
#include <p4/strarray.h>
#include <p4/spec.h>
#include <p4/mapapi.h>
#include "p4gospecmgr.h"
#include "p4goresult.h"
#include "p4gomergedata.h"
#include "p4goclientuser.h"
#include "p4goclientapi.h"
#include "p4go.h"
#include "p4gocallback.h"
#include "p4gospecdata.h"

const char*
P4Identify( P4GoClientApi* api )
{
    StrBuf s( "P4GoClientApi " );
    s << "2025.2"; // ToDo: Drive this off the cgo args?
    s << " P4API " << api->GetBuild();
    char* ret = (char*)malloc( s.Length() + 1 );
    strcpy( ret, s.Text() );
    return ret;
}

P4GoClientApi*
NewClientApi()
{
    P4GoClientApi* api = new P4GoClientApi;
    return api;
}

void
FreeClientApi( P4GoClientApi* api )
{
    delete api;
}

int
P4Connect( P4GoClientApi* api, Error* e )
{
    return api->Connect( e );
}

int
P4Connected( P4GoClientApi* api )
{
    return api->Connected();
}

int
P4Disconnect( P4GoClientApi* api, Error* e )
{
    return api->Disconnect( e );
}

void
Run( P4GoClientApi* api, char* cmd, int argc, char** argv, Error* e )
{
    api->Run( cmd, argc, argv, e );
}

int
ResultCount( P4GoClientApi* api )
{
    return api->GetResults()->Count();
}

int
ResultGet( P4GoClientApi* api, int index, int* type, P4GoResult** ret )
{

    *ret = (P4GoResult*)api->GetResults()->Get( index );
    if( *ret )
        *type = ( *ret )->type;
    return *ret != 0;
}

const char*
ResultGetString( P4GoResult* ret )
{
    if( ret->type == STRING || ret->type == TRACK ) {
        char* r = (char*)malloc( ret->str->Length() + 1 );
        strcpy( r, ret->str->Text() );
        return r;
    } else if( ret->type == ERROR ) {
        StrBuf msg;
        ret->err->Fmt( &msg, 0 );
        char* r = (char*)malloc( msg.Length() + 1 );
        strcpy( r, msg.Text() );
        return r;
    }
    return 0;
}

const char* 
ResultGetBinary( P4GoResult* ret, int* len )
{
    if( ret->type == BINARY ) {
        char* r = (char*)malloc( ret->str->Length() + 1 );
        strcpy( r, ret->str->Text() );
        *len = ret->str->Length();
        return r;
    }
    return 0;
}

Error*
ResultGetError( P4GoResult* ret )
{
    if( ret->type == ERROR ) {
        return ret->err;
    }
    return 0;
}

P4GoSpecData*
ResultGetSpec( P4GoResult* ret )
{
    if( ret->type == SPEC ) {
        return ret->spec;
    }
    return 0;
}

int
ResultGetKeyPair( P4GoResult* ret, int index, char** var, char** val )
{
    if( ret->type == DICT ) {
        StrRef svar, sval;
        if( ret->dict->GetVar( index, svar, sval ) ) {
            *var = svar.Text();
            *val = sval.Text();
            return 1;
        }
    } else if( ret->type == SPEC ) {
        StrRef svar, sval;
        if( ret->spec->GetDict()->GetVar( index, svar, sval ) ) {
            *var = svar.Text();
            *val = sval.Text();
            return 1;
        }
    }
    return 0;
}

int
IsIgnored( P4GoClientApi* api, char* path )
{
    return api->IsIgnored( path );
}

//
// Getters and Setters
//

int
GetApiLevel( P4GoClientApi* api )
{
    return api->GetApiLevel();
}

void
SetApiLevel( P4GoClientApi* api, int apiLevel )
{
    api->SetApiLevel( apiLevel );
}

int
GetStreams( P4GoClientApi* api )
{
    return api->IsStreams();
}

void
SetStreams( P4GoClientApi* api, int enableStreams )
{
    api->SetStreams( enableStreams );
}

int
GetTagged( P4GoClientApi* api )
{
    return api->IsTagged();
}

void
SetTagged( P4GoClientApi* api, int enableTagged )
{
    api->Tagged( enableTagged );
}

int
GetTrack( P4GoClientApi* api )
{
    return api->GetTrack();
}

int
SetTrack( P4GoClientApi* api, int enableTrack, Error* e )
{
    return api->SetTrack( enableTrack, e );
}

int
GetGraph( P4GoClientApi* api )
{
    return api->IsGraph();
}

void
SetGraph( P4GoClientApi* api, int enableGraph )
{
    api->SetGraph( enableGraph );
}

int
GetDebug( P4GoClientApi* api )
{
    return api->GetDebug();
}

void
SetDebug( P4GoClientApi* api, int debug )
{
    return api->SetDebug( debug );
}

const char*
GetCharset( P4GoClientApi* api )
{
    return api->GetCharset().Text();
}

int
SetCharset( P4GoClientApi* api, char* charset, Error* e )
{
    return api->SetCharset( charset, e );
}

const char*
GetCwd( P4GoClientApi* api )
{
    return api->GetCwd().Text();
}

void
SetCwd( P4GoClientApi* api, char* cwd )
{
    api->SetCwd( cwd );
}

const char*
GetClient( P4GoClientApi* api )
{
    return api->GetClient().Text();
}

void
SetClient( P4GoClientApi* api, char* client )
{
    return api->SetClient( client );
}

const char*
GetEnv( P4GoClientApi* api, char* env )
{
    return api->GetEnv( env );
}

int
SetEnv( P4GoClientApi* api, char* env, char* value, Error* e )
{
    return api->SetEnv( env, value, e );
}

const char*
GetEnviroFile( P4GoClientApi* api )
{
    const StrPtr* v = api->GetEnviroFile();
    return v ? v->Text() : 0;
}

void
SetEnviroFile( P4GoClientApi* api, char* enviroFile )
{
    api->SetEnviroFile( enviroFile );
}

const char*
GetEVar( P4GoClientApi* api, char* evar )
{
    const StrPtr* v = api->GetEVar( evar );
    return v ? v->Text() : 0;
}

void
SetEVar( P4GoClientApi* api, char* evar, char* value )
{
    return api->SetEVar( evar, value );
}

const char*
GetHost( P4GoClientApi* api )
{
    return api->GetHost().Text();
}

void
SetHost( P4GoClientApi* api, char* host )
{
    return api->SetHost( host );
}

const char*
GetIgnoreFile( P4GoClientApi* api )
{
    return api->GetIgnoreFile().Text();
}

void
SetIgnoreFile( P4GoClientApi* api, char* ignoreFile )
{
    return api->SetIgnoreFile( ignoreFile );
}

const char*
GetLanguage( P4GoClientApi* api )
{
    return api->GetLanguage().Text();
}

void
SetLanguage( P4GoClientApi* api, char* language )
{
    return api->SetLanguage( language );
}

const char*
GetP4ConfigFile( P4GoClientApi* api )
{
    return api->GetConfig().Text();
}

const char*
GetPassword( P4GoClientApi* api )
{
    return api->GetPassword().Text();
}

void
SetPassword( P4GoClientApi* api, char* password )
{
    api->SetPassword( password );
}

const char*
GetPort( P4GoClientApi* api )
{
    return api->GetPort().Text();
}

void
SetPort( P4GoClientApi* api, char* port )
{
    api->SetPort( port );
}

const char*
GetProg( P4GoClientApi* api )
{
    return api->GetProg().Text();
}

void
SetProg( P4GoClientApi* api, char* prog )
{
    api->SetProg( prog );
}

void
SetProtocol( P4GoClientApi* api, char* protocol, char* value )
{
    return api->SetProtocol( protocol, value );
}

void
SetVar( P4GoClientApi* api, char* variable, char* value )
{
    return api->SetVar( variable, value );
}

const char*
GetTicketFile( P4GoClientApi* api )
{
    return api->GetTicketFile().Text();
}

void
SetTicketFile( P4GoClientApi* api, char* ticketFile )
{
    api->SetTicketFile( ticketFile );
}

const char*
GetTrustFile( P4GoClientApi* api )
{
    return api->GetTrustFile().Text();
}

void
SetTrustFile( P4GoClientApi* api, char* trustFile )
{
    return api->SetTrustFile( trustFile );
}

const char*
GetUser( P4GoClientApi* api )
{
    return api->GetUser().Text();
}

void
SetUser( P4GoClientApi* api, char* user )
{
    api->SetUser( user );
}

const char*
GetP4Version( P4GoClientApi* api )
{
    return api->GetVersion().Text();
}

void
SetP4Version( P4GoClientApi* api, char* version )
{
    api->SetVersion( version );
}

int
GetMaxResults( P4GoClientApi* api )
{
    return api->GetMaxResults();
}

void
SetMaxResults( P4GoClientApi* api, int maxResults )
{
    api->SetMaxResults( maxResults );
}

int
GetMaxScanRows( P4GoClientApi* api )
{
    return api->GetMaxScanRows();
}

void
SetMaxScanRows( P4GoClientApi* api, int maxScanRows )
{
    api->SetMaxScanRows( maxScanRows );
}

int
GetMaxLockTime( P4GoClientApi* api )
{
    return api->GetMaxLockTime();
}

void
SetMaxLockTime( P4GoClientApi* api, int maxLockTime )
{
    return api->SetMaxLockTime( maxLockTime );
}

void
ResetInput( P4GoClientApi* api )
{
    api->ResetInput();
}

void
AppendInput( P4GoClientApi* api, char* input )
{
    api->AppendInput( input );
}

P4GoSpecData*
ParseSpec( P4GoClientApi* api, char* spec, char* form, Error* e )
{
    return api->ParseSpec( spec, form, e );
}

char*
FormatSpec( P4GoClientApi* api, char* spec, StrDict* dict, Error* e )
{
    return api->FormatSpec( spec, dict, e );
}

int
P4ServerLevel( P4GoClientApi* api, Error* e )
{
    return api->GetServerLevel(e);
}

int
P4ServerCaseSensitive( P4GoClientApi* api, Error* e )
{
    return api->ServerCaseSensitive(e);
}

int
P4ServerUnicode( P4GoClientApi* api, Error* e )
{
    return api->ServerUnicode(e);
}

//
// Error wrapper
//

Error* MakeError()
{
    return new Error;
}

void FreeError( Error* e )
{
    delete e;
}

const char*
FmtError( Error* e, int i )
{
    StrBuf buf;
    e->Fmt( i + 1, buf, 0 );
    char* ret = (char*)malloc( buf.Length() + 1 );
    strcpy( ret, buf.Text() );
    return ret;
}

int
GetErrorCode( Error* e, int i )
{
    return e->GetId( i )->code;
}

int
GetErrorCount( Error* e )
{
    return e->GetErrorCount();
}

int
GetErrorSeverity( Error* e )
{
    return e->GetSeverity();
}

int
GetErrorSeverityI( Error* e, int i )
{
    return e->GetId( i )->Severity();
}

StrDict*
GetDict( Error* e )
{
    return e->GetDict();
}

P4GoProgress*
NewProgress()
{
    P4GoProgress* progress = new P4GoProgress( cbProgressInit,
                                               cbProgressDescription,
                                               cbProgressTotal,
                                               cbProgressUpdate,
                                               cbProgressDone );
    return progress;
}

void
FreeProgress( P4GoProgress* progress )
{
    delete progress;
}

void
SetProgress( P4GoClientApi* api, P4GoProgress* progress )
{
    api->SetProgress( progress );
}

P4GoProgress*
GetProgress( P4GoClientApi* api )
{
    return api->GetProgress();
}

P4GoHandler*
NewHandler()
{
    return new P4GoHandler( cbHandleBinary,
                            cbHandleMessage,
                            cbHandleStat,
                            cbHandleText,
                            cbHandleTrack,
                            cbHandleSpec );
}

void
FreeHandler( P4GoHandler* handler )
{
    delete handler;
}

void
SetHandler( P4GoClientApi* api, P4GoHandler* handler )
{
    api->SetHandler( handler );
}

P4GoHandler*
GetHandler( P4GoClientApi* api )
{
    return api->GetHandler();
}

int
StrDictGetKeyPair( StrDict* dict, int index, char** var, char** val )
{
    StrRef svar, sval;
    if( dict->GetVar( index, svar, sval ) ) {
        *var = svar.Text();
        *val = sval.Text();
        return 1;
    }
    return 0;
}

P4GoSSOHandler*
NewSSOHandler()
{
    return new P4GoSSOHandler( cbSSOAuthorize );
}

void
FreeSSOHandler( P4GoSSOHandler* handler )
{
    delete handler;
}

void
SetSSOHandler( P4GoClientApi* api, P4GoSSOHandler* handler )
{
    api->SetSSOHandler( handler );
}

P4GoSSOHandler*
GetSSOHandler( P4GoClientApi* api )
{
    return api->GetSSOHandler();
}

void
StrDictSetKeyPair( StrDict* dict, char* var, char* val )
{
    dict->SetVar( var, val );
}

StrDict*
NewStrDict()
{
    return new StrBufDict;
}

void
FreeStrDict( StrDict* dict )
{
    delete dict;
}

int
SpecDataGetKeyPair( P4GoSpecData* spec, int index, char** var, char** val )
{
    StrRef svar, sval;
    if( spec->GetDict()->GetVar( index, svar, sval ) ) {
        *var = svar.Text();
        *val = sval.Text();
        return 1;
    }
    return 0;
}

void
FreeSpecData( P4GoSpecData* spec )
{
    delete spec;
}

void
SetResolveHandler( P4GoClientApi* api, P4GoResolveHandler* handler )
{
    api->SetResolveHandler( handler );
}

P4GoResolveHandler*
GetResolveHandler( P4GoClientApi* api )
{
    return api->GetResolveHandler();
}

P4GoResolveHandler*
NewResolveHandler()
{
    return new P4GoResolveHandler( cbResolve );
}

void
FreeResolveHandler( P4GoResolveHandler* handler )
{
    delete handler;
}

// MapApi

MapApi*
NewMapApi()
{
    return new MapApi();
}

void
FreeMapApi( MapApi* mapapi )
{
    delete mapapi;
}

MapApi*
JoinMapApi( MapApi* m1, MapApi* m2 )
{
    return MapApi::Join( m1, m2 );
}

void
MapApiInsert( MapApi* mapapi, char* lhs, char* rhs, int flag )
{
    if( rhs && strlen( rhs ) )
        mapapi->Insert( StrRef( lhs ), StrRef( rhs ), (MapType)flag );
    else
        mapapi->Insert( StrRef( lhs ), (MapType)flag );
}

void
MapApiClear( MapApi* mapapi )
{
    mapapi->Clear();
}

int
MapApiCount( MapApi* mapapi )
{
    return mapapi->Count();
}

MapApi*
MapApiReverse( MapApi* mapapi )
{
    MapApi* nmap = new MapApi;
    const StrPtr* l;
    const StrPtr* r;
    MapType t;

    for( int i = 0; i < mapapi->Count(); i++ ) {
        l = mapapi->GetLeft( i );
        r = mapapi->GetRight( i );
        t = mapapi->GetType( i );

        nmap->Insert( *r, *l, t );
    }

    delete mapapi;
    return nmap;
}

char*
MapApiLhs( MapApi* mapapi, int i )
{
    const StrPtr* s = mapapi->GetLeft( i );
    return s ? s->Text() : 0;
}

char*
MapApiRhs( MapApi* mapapi, int i )
{
    const StrPtr* s = mapapi->GetRight( i );
    return s ? s->Text() : 0;
}

int
MapApiType( MapApi* mapapi, int i )
{
    return mapapi->GetType( i );
}

char*
MapApiTranslate( MapApi* mapapi, char* input, int dir )
{
    StrBuf out;
    if( mapapi->Translate( StrRef( input ), out, (MapDir)dir ) ) {
        char* cout = (char*)malloc( out.Length() + 1 );
        out.StrCpy( cout );
        return cout;
    }
    return 0;
}

char**
MapApiTranslateArray( MapApi* mapapi, char* input, int dir, int* results )
{
    StrArray arr;
    if( mapapi->Translate( StrRef( input ), arr, (MapDir)dir ) ) {
        *results = arr.Count();
        char** carr = (char**)malloc( sizeof( char* ) * ( *results + 1 ) );
        for( int i = 0; i < *results; i++ ) {
            const StrPtr* out = arr.Get( i );
            carr[i] = (char*)malloc( out->Length() + 1 );
            out->StrCpy( carr[i] );
        }
        carr[*results] = 0;
        return carr;
    }
    return 0;
}

//
// P4GoMergeData wrapper
//

char*
MergeDataGetYourName( P4GoMergeData* m )
{
    return m->GetYourName();
}

char*
MergeDataGetTheirName( P4GoMergeData* m )
{
    return m->GetTheirName();
}

char*
MergeDataGetBaseName( P4GoMergeData* m )
{
    return m->GetBaseName();
}

char*
MergeDataGetYourPath( P4GoMergeData* m )
{
    return m->GetYourPath();
}

char*
MergeDataGetTheirPath( P4GoMergeData* m )
{
    return m->GetTheirPath();
}

char*
MergeDataGetBasePath( P4GoMergeData* m )
{
    return m->GetBasePath();
}

char*
MergeDataGetResultPath( P4GoMergeData* m )
{
    return m->GetResultPath();
}

int
MergeDataRunMergeTool( P4GoMergeData* m )
{
    return m->RunMergeTool();
}

int
MergeDataGetActionResolveStatus( P4GoMergeData* m )
{
    return m->GetActionResolveStatus();
}

int
MergeDataGetContentResolveStatus( P4GoMergeData* m )
{
    return m->GetContentResolveStatus();
}

void*
MergeDataGetMergeInfo( P4GoMergeData* m )
{
    return m->GetMergeInfo();
}

const Error*
MergeDataGetMergeAction( P4GoMergeData* m )
{
    return m->GetMergeAction();
}

const Error*
MergeDataGetYoursAction( P4GoMergeData* m )
{
    return m->GetYoursAction();
}

const Error*
MergeDataGetTheirAction( P4GoMergeData* m )
{
    return m->GetTheirAction();
}

const Error*
MergeDataGetType( P4GoMergeData* m )
{
    return m->GetType();
}

char*
MergeDataGetString( P4GoMergeData* m )
{
    return m->GetString();
}

int
MergeDataGetMergeHint( P4GoMergeData* m )
{
    return m->GetMergeHint();
}

int SpecDataGetVar(P4GoSpecData* sd, const char* key, char** value)
{
    if( !sd ) return 0;
    
    StrDict* dict = sd->GetDict();
    StrPtr* val = dict->GetVar( key );
    
    if( !val ) return 0;
    
    *value = strdup( val->Text() );
    return 1;
}

int SpecDataGetVarCount(P4GoSpecData* sd)
{
    if( !sd ) return 0;
    
    StrDict* dict = sd->GetDict();
    return dict->GetCount();
}

int SpecDataGetKey(P4GoSpecData* sd, int index, char** key)
{
    if( !sd ) return 0;
    
    StrDict* dict = sd->GetDict();
    StrRef var, val;
    
    if( !dict->GetVar( index, var, val ) ) return 0;
    
    *key = strdup( var.Text() );
    return 1;
}

int SpecDataIsArray(P4GoSpecData* sd, const char* key)
{
    if( !sd ) return 0;
    
    // Check if there's a key with "[0]" suffix
    StrBuf arrayKey;
    arrayKey << key << "[0]";
    
    StrDict* dict = sd->GetDict();
    StrPtr* val = dict->GetVar( arrayKey );
    
    return val != NULL ? 1 : 0;
}

int SpecDataGetArraySize(P4GoSpecData* sd, const char* key)
{
    if( !sd ) return 0;
    
    StrDict* dict = sd->GetDict();
    int count = 0;
    
    // Count array elements
    while( true )
    {
        StrBuf arrayKey;
        arrayKey << key << "[" << count << "]";
        
        StrPtr* val = dict->GetVar( arrayKey );
        if( !val ) break;
        
        count++;
    }
    
    return count;
}

int SpecDataGetArrayValue(P4GoSpecData* sd, const char* key, int index, char** value)
{
    if( !sd ) return 0;
    
    StrBuf arrayKey;
    arrayKey << key << "[" << index << "]";
    
    StrDict* dict = sd->GetDict();
    StrPtr* val = dict->GetVar( arrayKey );
    
    if( !val ) return 0;
    
    *value = strdup( val->Text() );
    return 1;
}