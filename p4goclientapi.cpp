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
#include <p4/strtable.h>
#include <p4/vararray.h>
#include <p4/i18napi.h>
#include <p4/enviro.h>
#include <p4/hostenv.h>
#include <p4/spec.h>
#include <p4/ignore.h>
#include <p4/debug.h>
#include "p4gospecmgr.h"
#include "p4gospecdata.h"
#include "p4goresult.h"
#include "p4gomergedata.h"
#include "p4goclientuser.h"
#include "p4goclientapi.h"

P4GoClientApi::P4GoClientApi()
  : ui( &specMgr )
{
    debug = 0;
    server2 = 0;
    depth = 0;
    exceptionLevel = 2;
    maxResults = 0;
    maxScanRows = 0;
    maxLockTime = 0;
    InitFlags();
    apiLevel = atoi( P4Tag::l_client );
    enviro = new Enviro;
    prog = "Unnamed P4Go program";

    client.SetProtocol( "specstring", "" );
    client.SetBreak( &ui );

    //
    // Load any P4CONFIG file
    //
    HostEnv henv;
    StrBuf cwd;

    henv.GetCwd( cwd, enviro );
    if( cwd.Length() )
        enviro->Config( cwd );
}

P4GoClientApi::~P4GoClientApi()
{
    if( IsConnected() ) {
        Error e;
        client.Final( &e );
        // Ignore errors
    }
    delete enviro;
}

const char*
P4GoClientApi::GetEnv( const char* v )
{
    return enviro->Get( v );
}

void
P4GoClientApi::SetEnviroFile( const char* c )
{
    enviro->SetEnviroFile( c );
}

const StrPtr*
P4GoClientApi::GetEnviroFile()
{
    return enviro->GetEnviroFile();
}

void
P4GoClientApi::SetEVar( const char* var, const char* val )
{
    StrRef sVar( var );
    StrRef sVal( val );
    client.SetEVar( sVar, sVal );
}

const StrPtr*
P4GoClientApi::GetEVar( const char* var )
{
    StrRef sVar( var );
    return client.GetEVar( sVar );
}

void
P4GoClientApi::SetApiLevel( int level )
{
    StrBuf b;
    b << level;
    apiLevel = level;
    client.SetProtocol( "api", b.Text() );
    ui.SetApiLevel( level );
}

int
P4GoClientApi::SetCharset( const char* c, Error* e )
{
    StrRef cs_none( "none" );

    if( debug > 0 )
        fprintf( stderr, "[P4] Setting charset: %s\n", c );

    if( c && cs_none != c ) {
        CharSetApi::CharSet cs = CharSetApi::Lookup( c );
        if( cs < 0 ) {
            e->Set(E_FAILED, "P4#charset - Unknown or unsupported charset: %charset%" ) << c;
            return 0;
        }
        CharSetApi::CharSet utf8 = CharSetApi::Lookup( "utf8" );
        client.SetTrans( utf8, cs, utf8, utf8 );
        client.SetCharset( c );
    } else {
        // Disables automatic unicode detection if called
        // prior to init (2014.2)
        client.SetTrans( 0 );
    }
    return 1;
}

void
P4GoClientApi::SetCwd( const char* c )
{
    client.SetCwd( c );
    enviro->Config( StrRef( c ) );
}

void
P4GoClientApi::SetTicketFile( const char* p )
{
    client.SetTicketFile( p );
    ticketFile = p;
}

void
P4GoClientApi::SetTrustFile( const char* p )
{
    client.SetTrustFile( p );
    trustFile = p;
}

const StrPtr&
P4GoClientApi::GetTicketFile()
{
    if( ticketFile.Length() )
        return ticketFile;

    //
    // Load the current ticket file. Start with the default, and then
    // override it if P4TICKETS is set.
    //
    const char* t;

    HostEnv henv;
    henv.GetTicketFile( ticketFile );

    if( ( t = enviro->Get( "P4TICKETS" ) ) )
        ticketFile = t;

    return ticketFile;
}

const StrPtr&
P4GoClientApi::GetTrustFile()
{
    if( trustFile.Length() )
        return trustFile;

    //
    // Load the current trust file. Start with the default, and then
    // override it if P4TRUST is set.
    //
    const char* t;

    HostEnv henv;
    henv.GetTrustFile( trustFile );

    if( ( t = enviro->Get( "P4TICKETS" ) ) )
        trustFile = t;

    return trustFile;
}

void
P4GoClientApi::SetDebug( int d )
{
    debug = d;
    ui.SetDebug( d );
    specMgr.SetDebug( d );

    if( debug > 8 )
        p4debug.SetLevel( "rpc=5" );
    else
        p4debug.SetLevel( "rpc=0" );

    if( debug > 10 )
        p4debug.SetLevel( "ssl=3" );
    else
        p4debug.SetLevel( "ssl=0" );
}

void
P4GoClientApi::SetArrayConversion( int i )
{
    specMgr.SetArrayConversion( i );
}

void
P4GoClientApi::SetProtocol( const char* var, const char* val )
{
    client.SetProtocol( var, val );
}

void
P4GoClientApi::SetVar( const char* var, const char* val )
{
    client.SetVar( var, val );
}

int
P4GoClientApi::SetEnv( const char* var, const char* val, Error* e )
{

    enviro->Set( var, val, e );
    if( e->Test() && exceptionLevel ) {
        StrBuf m;
        e->Fmt( &m );
        e->Set(E_FAILED, "P4#set_env - %msg%") << m.Text();
    }

    if( e->Test() )
        return 0;

    // Fixes an issue on OS X where the next enviro->Get doesn't return the
    // cached void*
    enviro->Reload();

    return 1;
}

//
// connect to the Perforce server.
//

int
P4GoClientApi::Connect( Error* e )
{
    if( debug > 0 )
        fprintf( stderr, "[P4] Connecting to Perforce\n" );

    if( IsConnected() ) {
        e->Set(E_WARN, "P4#connect - Perforce client already connected!" );
        return 1;
    }

    return ConnectOrReconnect( e );
}

int
P4GoClientApi::ConnectOrReconnect( Error* e )
{
    
    if( IsTrackMode() )
        client.SetProtocol( "track", "" );

    ResetFlags();
    client.Init( e );
    if( e->Test() && exceptionLevel )
        e->Set(E_FAILED, "P4#connect - Failed to connect to Perforce server.");

    if( e->Test() )
        return 0;

    // If a handler is defined, reset the break functionality
    // for the KeepAlive function

    if( ui.GetHandler() != NULL )
    {
        client.SetBreak( &ui );
    }
    
    SetConnected();
    return 1;
}

//
// Disconnect session
//
int
P4GoClientApi::Disconnect( Error* e )
{
    if( debug > 0 )
        fprintf( stderr, "[P4] Disconnect\n" );

    if( !IsConnected()) {
        e->Set(E_WARN,  "P4#disconnect - not connected" );
        return 1;
    }
    
    client.Final( e );
    ResetFlags();

    // Clear the specdef cache.
    specMgr.Reset();

    // Clear out any results from the last command
    ui.Reset();

    return 1;
}

//
// Test whether or not connected
//
int
P4GoClientApi::Connected()
{
    if( IsConnected() && !client.Dropped() )
        return 1;
    else if( IsConnected() )
    {
        Error e;
        Disconnect( &e );
    }
    return 0;
}

void
P4GoClientApi::Tagged( int enable )
{
    if( enable )
        SetTag();
    else
        ClearTag();
}

int
P4GoClientApi::SetTrack( int enable, Error* e )
{
    if( IsConnected() ) {
        if( exceptionLevel ) {
            e->Set(E_FAILED, "P4#track - Can't change performance tracking once you've connected." );
        }
        return 0;
    } else if( enable ) {
        SetTrackMode();
        ui.SetTrack( true );
    } else {
        ClearTrackMode();
        ui.SetTrack( false );
    }
    return 1;
}

void
P4GoClientApi::SetStreams( int enable )
{
    if( enable )
        SetStreamsMode();
    else
        ClearStreamsMode();
}

void
P4GoClientApi::SetGraph( int enable )
{
    if( enable )
        SetGraphMode();
    else
        ClearGraphMode();
}

int
P4GoClientApi::GetServerLevel(Error* e)
{
    if( !IsConnected() )
        e->Set( E_FAILED, "ServerLevel - Not connected to a Perforce Server." );
    if( !IsCmdRun() )
        Run( "info", 0, 0, e );
    return server2;
}

int
P4GoClientApi::ServerCaseSensitive(Error* e)
{
    if( !IsConnected() )
        e->Set( E_FAILED, "ServerCaseSensitive - Not connected to a Perforce Server." );
    if( !IsCmdRun() )
        Run( "info", 0, 0, e );
    return !IsCaseFold();
}

int
P4GoClientApi::ServerUnicode(Error* e)
{
    if( !IsConnected() )
        e->Set( E_FAILED, "ServerUnicode - Not connected to a Perforce Server." );
    if( !IsCmdRun() )
        Run( "info", 0, 0, e );
    return IsUnicode();
}

// Check if the supplied path falls within the view of the ignore file
int
P4GoClientApi::IsIgnored( const char* path )
{
    Ignore* ignore = client.GetIgnore();
    if( !ignore )
        return 0;

    StrRef p( path );
    return ignore->Reject( p, client.GetIgnoreFile() );
}

//
// Run returns the results of the command. If the client has not been
// connected, then an exception is raised but errors from Perforce
// commands are returned via the Errors() and ErrorCount() interfaces
// and not via exceptions because one failure in a command applied to many
// files would interrupt processing of all the other files if an exception
// is raised.
//

P4GoResults*
P4GoClientApi::Run( const char* cmd, int argc, char* const* argv, Error* e )
{
    // Save the entire command string for our error messages. Makes it
    // easy to see where a script has gone wrong.
    StrBuf cmdString;
    cmdString << "\"p4 " << cmd;
    for( int i = 0; i < argc; i++ )
        cmdString << " " << argv[i];
    cmdString << "\"";

    if( debug > 0 )
        fprintf( stderr, "[P4] Executing %s\n", cmdString.Text() );

    if( depth ) {
        e->Set(E_WARN, "P4#run - Can't execute nested Perforce commands." );
        return 0;
    }

    // Clear out any results from the previous command
    ui.Reset();

    if( !IsConnected() && exceptionLevel )
        e->Set( E_FAILED, "P4#run - Not connected to a Perforce Server." );

    if( !IsConnected() )
        return 0;

    // Tell the UI which command we're running.
    ui.SetCommand( cmd );

    depth++;
    RunCmd( cmd, &ui, argc, argv );
    depth--;

    if( ui.GetHandler() != NULL ) {
        if( client.Dropped() && !ui.IsAlive() ) {
            Disconnect( e );
            ConnectOrReconnect( e );
        }
    }

    P4GoResults* results = ui.GetResults();

    return results;
}

void
P4GoClientApi::RunCmd( const char* cmd,
                       ClientUser* ui,
                       int argc,
                       char* const* argv )
{
    client.SetProg( &prog );
    if( version.Length() )
        client.SetVersion( &version );

    if( IsTag() )
        client.SetVar( "tag" );

    if( IsStreams() && apiLevel > 69 )
        client.SetVar( "enableStreams", "" );

    if( IsGraph() && apiLevel > 81 )
        client.SetVar( "enableGraph", "" );

    // If maxresults or maxscanrows is set, enforce them now
    if( maxResults )
        client.SetVar( "maxResults", maxResults );
    if( maxScanRows )
        client.SetVar( "maxScanRows", maxScanRows );
    if( maxLockTime )
        client.SetVar( "maxLockTime", maxLockTime );

    //	If progress is set, set progress var.
    if( ((P4GoClientUser*)ui)->GetProgress() != NULL )
        client.SetVar( P4Tag::v_progress, 1 );

    client.SetArgv( argc, argv );
    client.Run( cmd, ui );

    // Can only read the protocol block *after* a command has been run.
    // Do this once only.
    if( !IsCmdRun() ) {
        StrPtr* s = 0;
        if( ( s = client.GetProtocol( P4Tag::v_server2 ) ) )
            server2 = s->Atoi();

        if( ( s = client.GetProtocol( P4Tag::v_unicode ) ) )
            if( s->Atoi() )
                SetUnicode();

        if( ( s = client.GetProtocol( P4Tag::v_nocase ) ) )
            SetCaseFold();
    }
    SetCmdRun();
}

//
// Parses a string supplied by the user into a hash. To do this we need
// the specstring from the server. We try to cache those as we see them,
// but the user may not have executed any commands to allow us to cache
// them so we may have to fetch the spec first.
//

P4GoSpecData*
P4GoClientApi::ParseSpec( const char* type, const char* form, Error* e )
{
    if( !specMgr.HaveSpecDef( type ) ) {
        if( exceptionLevel ) {
            e->Set( E_FAILED, "No spec definition for %type% objects." ) << type;
        }
        return 0;
    }

    // Got a specdef so now we can attempt to parse it.
    P4GoSpecData* spec = specMgr.StringToSpec( type, form, e );

    if( e->Test() ) {
        delete spec;
        if ( exceptionLevel ) {
            StrBuf m;
            e->Fmt( &m );
            e->Set( E_FAILED, "P4#parse_spec - %msg%" ) << m.Text();
        }
        return 0;
    }

    return spec;
}

//
// Converts a dict supplied by the user into a string using the specstring
// from the server. We may have to fetch the specstring first.
//

char*
P4GoClientApi::FormatSpec( const char* type, P4GoSpecData* spec, Error* e )
{
    
    if( !specMgr.HaveSpecDef( type ) ) {
        if( exceptionLevel ) {
            e->Set( E_FAILED, "No spec definition for %type% objects." ) << type;
        }
        return 0;
    }

    // Got a specdef so now we can attempt to convert.
    StrBuf buf;

    specMgr.SpecToString( type, spec, buf, e );
    if( !e->Test() ) {
        char* cbuf = (char*)malloc( buf.Length() + 1 );
        buf.StrCpy( cbuf );
        return cbuf;
    }

    if( exceptionLevel ) {
        StrBuf m;
        m = "Error converting hash to a string.";
        if( e->Test() )
            e->Fmt( m, EF_PLAIN );
        e->Set( E_FAILED, "P4#format_spec - %msg%" ) << m.Text();
    }
    return 0;
}

char*
P4GoClientApi::FormatSpec( const char* type, StrDict* dict, Error* e )
{
    P4GoSpecData spec( dict );
    return FormatSpec( type, &spec, e );
}

//
// Returns a hash whose keys contain the names of the fields in a spec of the
// specified type. 
//
StrDict*
P4GoClientApi::SpecFields( const char* type, Error* e )
{
    if( !specMgr.HaveSpecDef( type ) ) {
        if( exceptionLevel ) {
            e->Set( E_FAILED, "No spec definition for %type% objects." ) << type;
        }
        return 0;
    }

    return specMgr.SpecFields( type );
}
