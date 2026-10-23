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
#include <p4/clientprog.h>
#include <p4/i18napi.h>
#include <p4/charcvt.h>
#include <p4/diff.h>
#include <p4/vararray.h>
#include <p4/strtable.h>
#include <p4/strarray.h>
#include <p4/spec.h>
#include "p4gomergedata.h"
#include "p4gospecmgr.h"
#include "p4goresult.h"
#include "p4goclientuser.h"
#include "p4godebug.h"

//
// Progress callbacks
//

class P4GoClientProgress : public ClientProgress
{
  public:
    P4GoClientProgress( P4GoProgress* prog, int t );
    virtual ~P4GoClientProgress();

  public:
    void Description( const StrPtr* d, int u );
    void Total( long t );
    int Update( long update );
    void Done( int f );

  private:
    P4GoProgress* progress;
};

P4GoClientProgress::P4GoClientProgress( P4GoProgress* prog, int type )
{
    progress = prog;
    progress->Init( type );
}

P4GoClientProgress::~P4GoClientProgress() {}

void
P4GoClientProgress::Description( const StrPtr* desc, int units )
{
    progress->Description( desc, units );
}

void
P4GoClientProgress::Total( long total )
{
    progress->Total( total );
}

int
P4GoClientProgress::Update( long position )
{
    progress->Update( position );
    return 0;
}

void
P4GoClientProgress::Done( int fail )
{
    progress->Done( fail );
}

P4GoProgress::P4GoProgress( cbInit_t cbInit,
                            cbDesc_t cbDesc,
                            cbTotal_t cbTotal,
                            cbUpdate_t cbUpdate,
                            cbDone_t cbDone )
  : cbInit( cbInit )
  , cbDesc( cbDesc )
  , cbTotal( cbTotal )
  , cbUpdate( cbUpdate )
  , cbDone( cbDone )
{
}

void
P4GoProgress::Init( int type )
{
    cbInit( this, type );
}

void
P4GoProgress::Description( const StrPtr* desc, int units )
{
    cbDesc( this, desc->Text(), units );
}

void
P4GoProgress::Total( long total )
{
    cbTotal( this, total );
}

void
P4GoProgress::Update( long position )
{
    cbUpdate( this, position );
}

void
P4GoProgress::Done( int fail )
{
    cbDone( this, fail );
}

P4GoKeepAlive::P4GoKeepAlive( cbIsAlive_t cbIsAlive )
  : cbIsAlive( cbIsAlive )
{
}

int
P4GoKeepAlive::IsAlive()
{
    return cbIsAlive( this );
}

P4GoHandler::P4GoHandler( cbHandleBin_t cbHandleBin,
                          cbHandleMsg_t cbHandleMsg,
                          cbHandleStat_t cbHandleStat,
                          cbHandleText_t cbHandleText,
                          cbHandleTrack_t cbHandleTrack,
                          cbHandleSpec_t cbHandleSpec )
  : cbHandleBin( cbHandleBin )
  , cbHandleMsg( cbHandleMsg )
  , cbHandleStat( cbHandleStat )
  , cbHandleText( cbHandleText )
  , cbHandleTrack( cbHandleTrack )
  , cbHandleSpec( cbHandleSpec )
{
}

int
P4GoHandler::HandleBinary( StrPtr data )
{
    return cbHandleBin( this, data.Text(), data.Length() );
}

int
P4GoHandler::HandleMessage( Error* e )
{
    return cbHandleMsg( this, e );
}

int
P4GoHandler::HandleStat( StrDict* d )
{
    return cbHandleStat( this, d );
}

int
P4GoHandler::HandleText( StrPtr data )
{
    return cbHandleText( this, data.Text() );
}

int
P4GoHandler::HandleTrack( StrPtr data )
{
    return cbHandleTrack( this, data.Text() );
}

int
P4GoHandler::HandleSpec( P4GoSpecData* spec )
{
    return cbHandleSpec( this, spec );
}

P4GoSSOHandler::P4GoSSOHandler( cbSSOAuthorize_t cbSSOAuthorize )
  : cbSSOAuthorize( cbSSOAuthorize )
{
}

ClientSSOStatus
P4GoSSOHandler::Authorize( StrDict& vars, int maxLength, StrBuf& result )
{
    result.Clear();
    char* rp = 0;
    int ret = cbSSOAuthorize( this, &vars, maxLength, &rp );
    if( rp ) {
        result.Set( rp );
        free( rp );
    }
    return (ClientSSOStatus)ret;
}

P4GoResolveHandler::P4GoResolveHandler( cbResolve_t cbResolve )
  : cbResolve( cbResolve )
{
}

int
P4GoResolveHandler::Resolve( P4GoMergeData* m )
{
    return cbResolve( this, m );
}

P4GoFileSysBuffer::P4GoFileSysBuffer( cbFbRelease_t cbFbRelease,
                                      cbFbOpen_t cbFbOpen,
                                      cbFbWrite_t cbFbWrite,
                                      cbFbRead_t cbFbRead,
                                      cbFbClose_t cbFbClose,
                                      cbFbSeek_t cbFbSeek,
                                      cbFbSizeHint_t cbFbSizeHint )
  : cbFbRelease( cbFbRelease )
  , cbFbOpen( cbFbOpen )
  , cbFbWrite( cbFbWrite )
  , cbFbRead( cbFbRead )
  , cbFbClose( cbFbClose )
  , cbFbSeek( cbFbSeek )
  , cbFbSizeHint( cbFbSizeHint )
{
}

void
P4GoFileSysBuffer::Release()
{
    cbFbRelease( this );
}

void
P4GoFileSysBuffer::Open( StrPtr *path, FileOpenMode mode, Error *e )
{
    cbFbOpen( this, path->Text(), (int)mode, e );
}

void
P4GoFileSysBuffer::Write( const char *buf, int len, Error *e )
{
    cbFbWrite( this, (char*)buf, len, e );
}

int
P4GoFileSysBuffer::Read( char *buf, int len, Error *e )
{
    return cbFbRead( this, buf, len, e );
}

void
P4GoFileSysBuffer::Close( Error *e )
{
    cbFbClose( this, e );
}

void
P4GoFileSysBuffer::Seek( offL_t offset, Error* e )
{
    cbFbSeek( this, offset, e );
}

void
P4GoFileSysBuffer::SizeHint( offL_t size )
{
    cbFbSizeHint( this, size );
}

P4GoClientUser::P4GoClientUser( P4GoSpecMgr* s )
{
    specMgr = s;
    debug = 0;
    apiLevel = atoi( P4Tag::l_client );
    input = new StrArray();
    handler = 0;
    resolveHandler = 0;
    progress = 0;
    keepAlive = 0;
    fileBuffer = 0;
    alive = 1;
    track = false;
    oldPasswordPromptSeen = false;
    discardUnusedOldPasswordSlot = false;
}

P4GoClientUser::~P4GoClientUser()
{
    delete input;
}

void
P4GoClientUser::Reset()
{
    results.Reset();
    // Leave input alone.

    alive = 1;
    oldPasswordPromptSeen = false;
}

void
P4GoClientUser::SetApiLevel( int l )
{
    apiLevel = l;
    results.SetApiLevel( l );
}

void
P4GoClientUser::Finished()
{
    // Reset input coz we should be done with it now.
    if( P4GODB_CALLS && input->Count() )
        fprintf( stderr, "[P4] Cleaning up saved input\n" );

    input->Clear();
}

/*
 * Handling of output
 */

bool
P4GoClientUser::CallOutputMethod( StrPtr data, bool binary )
{
    if( P4GODB_COMMANDS )
        fprintf(
          stderr, "[P4] CallOutputMethod(%s)\n", binary ? "binary" : "text" );

    int ret =
      binary ? handler->HandleBinary( data ) : handler->HandleText( data );

    if( P4GODB_COMMANDS )
        fprintf( stderr, "[P4] CallOutputMethod returned %d\n", ret );

    if( ret == 2 ) {
        if( P4GODB_COMMANDS )
            fprintf( stderr, "[P4] CallOutputMethod cancelled\n" );
        alive = 0;
    }

    return ( ret == 0 );
}

bool
P4GoClientUser::CallOutputMethod( StrDict* data )
{
    if( P4GODB_COMMANDS )
        fprintf( stderr, "[P4] CallOutputMethod(dict)\n" );

    int ret = handler->HandleStat( data );

    if( P4GODB_COMMANDS )
        fprintf( stderr, "[P4] CallOutputMethod returned %d\n", ret );

    if( ret == 2 ) {
        if( P4GODB_COMMANDS )
            fprintf( stderr, "[P4] CallOutputMethod cancelled\n" );
        alive = 0;
    }

    return ( ret == 0 );
}

bool
P4GoClientUser::CallOutputMethod( Error* e )
{
    if( P4GODB_COMMANDS )
        fprintf( stderr, "[P4] CallOutputMethod(error)\n" );

    int ret = handler->HandleMessage( e );

    if( P4GODB_COMMANDS )
        fprintf( stderr, "[P4] CallOutputMethod returned %d\n", ret );

    if( ret == 2 ) {
        if( P4GODB_COMMANDS )
            fprintf( stderr, "[P4] CallOutputMethod cancelled\n" );
        alive = 0;
    }

    return ( ret == 0 );
}

bool
P4GoClientUser::CallOutputMethod( P4GoSpecData* data )
{
    if( P4GODB_COMMANDS )
        fprintf( stderr, "[P4] CallOutputMethod(spec)\n" );

    int ret = 0; // handler->HandleSpec( data );

    if( P4GODB_COMMANDS )
        fprintf( stderr, "[P4] CallOutputMethod returned %d\n", ret );

    if( ret == 2 ) {
        if( P4GODB_COMMANDS )
            fprintf( stderr, "[P4] CallOutputMethod cancelled\n" );
        alive = 0;
    }

    return ( ret == 0 );
}

void
P4GoClientUser::ProcessOutput( StrPtr data, bool binary )
{
    if( this->handler ) {
        if( CallOutputMethod( data, binary ) )
            results.AddOutput( data, binary );
    } else
        results.AddOutput( data, binary );
}

void
P4GoClientUser::ProcessOutput( StrDict* data )
{
    if( this->handler ) {
        if( CallOutputMethod( data ) )
            results.AddOutput( data );
        else
            delete data;
    } else
        results.AddOutput( data );
}

void
P4GoClientUser::ProcessOutput( P4GoSpecData* data )
{
    if( this->handler ) {
        if( CallOutputMethod( data ) )
            results.AddOutput( data );
    } else
        results.AddOutput( data );
}

void
P4GoClientUser::ProcessMessage( Error* e )
{
    if( this->handler ) {
        if( CallOutputMethod( e ) )
            results.AddOutput( e );
    } else
        results.AddOutput( e );
}

void
P4GoClientUser::OutputInfo( char level, const char* data )
{
    if( P4GODB_CALLS )
        fprintf( stderr, "[P4] OutputInfo()\n" );
    if( P4GODB_DATA )
        fprintf( stderr, "... [%c] %s\n", level, data );

    ProcessOutput( StrRef( data ), false );
}

/*
 * Very little should use this. Most output arrives via
 * Message() these days, but -Ztrack output, and a few older
 * anachronisms might take this route.
 */

void
P4GoClientUser::OutputText( const char* data, int length )
{
    if( P4GODB_CALLS )
        fprintf( stderr, "[P4] OutputText()\n" );
    if( P4GODB_DATA )
        fprintf( stderr, "... [%d]%*s\n", length, length, data );
    if( track && length > 4 && data[0] == '-' && data[1] == '-' &&
        data[2] == '-' && data[3] == ' ' ) {
        int p = 4;
        for( int i = 4; i < length; ++i ) {
            if( data[i] == '\n' ) {
                if( i > p ) {
                    results.AddTrack( StrRef( data + p, i - p ) );
                    p = i + 5;
                } else {
                    // this was not track data after all,
                    // try to rollback the damage done
                    ProcessOutput( StrRef( data, length ), false );
                    results.DeleteTrack();
                    return;
                }
            }
        }
    } else
        ProcessOutput( StrRef( data, length ), false );
}

void
P4GoClientUser::Message( Error* e )
{
    if( P4GODB_CALLS )
        fprintf( stderr, "[P4] Message()\n" );

    if( P4GODB_DATA ) {
        StrBuf t;
        e->Fmt( t, EF_PLAIN );
        fprintf( stderr, "... [%s] %s\n", e->FmtSeverity(), t.Text() );
    }

    ProcessMessage( e );
}

void
P4GoClientUser::OutputBinary( const char* data, int length )
{
    if( P4GODB_CALLS )
        fprintf( stderr, "[P4] OutputBinary()\n" );
    if( P4GODB_DATA ) {
        for( int l = 0; l < length; l++ ) {
            if( l % 16 == 0 )
                fprintf( stderr, "%s... ", l ? "\n" : "" );
            fprintf( stderr, "%#hhx ", data[l] );
        }
    }

    //
    // Binary is just stored in a string. Since the char * version of
    // P4Result::AddOutput() assumes it can strlen() to find the length,
    // we'll make the String object here.
    //
    ProcessOutput( StrRef( data, length ), true );
}

void
P4GoClientUser::HandleError( Error* e )
{
    if( P4GODB_CALLS )
        fprintf( stderr, "[P4] HandleError()\n" );

    if( P4GODB_DATA ) {
        StrBuf t;
        e->Fmt( t, EF_PLAIN );

        fprintf( stderr, "... [%s] %s\n", e->FmtSeverity(), t.Text() );
    }

    ProcessMessage( e );
}

void
P4GoClientUser::OutputStat( StrDict* values )
{
    StrPtr* spec = values->GetVar( "specdef" );
    StrPtr* data = values->GetVar( "data" );
    StrPtr* sf = values->GetVar( "specFormatted" );
    StrDict* dict = values;
    SpecDataTable specData;
    Error e;

    //
    // Determine whether or not the data we've got contains a spec in one form
    // or another. 2000.1 -> 2005.1 servers supplied the form in a data variable
    // and we use the spec variable to parse the form. 2005.2 and later servers
    // supply the spec ready-parsed but set the 'specFormatted' variable to tell
    // the client what's going on. Either way, we need the specdef variable set
    // to enable spec parsing.
    //
    int isspec = spec && ( sf || data );

    //
    // Save the spec definition for later
    //
    if( spec )
        specMgr->AddSpecDef( cmd.Text(), spec->Text() );

    //
    // Parse any form supplied in the 'data' variable and convert it into a
    // dictionary.
    //
    if( spec && data ) {
        // 2000.1 -> 2005.1 server's handle tagged form output by supplying the
        // form as text in the 'data' variable. We need to convert it to a
        // dictionary using the supplied spec.
        if( P4GODB_CALLS )
            fprintf( stderr, "[P4] OutputStat() - parsing form\n" );

        // Parse the form. Use the ParseNoValid() interface to prevent
        // errors caused by the use of invalid defaults for select items in
        // jobspecs.

        Spec s( spec->Text(), "", &e );

        if( !e.Test() )
            s.ParseNoValid( data->Text(), &specData, &e );
        if( e.Test() ) {
            HandleError( &e );
            return;
        }
        dict = specData.Dict();
    }

    //
    // If what we've got is a parsed form, then we'll convert it to a P4::Spec
    // object. Otherwise it's a plain hash.
    //
    if( isspec ) {
        if( P4GODB_CALLS )
            fprintf( stderr,
                     "[P4] OutputStat() - Converting to P4::Spec object\n" );
        ProcessOutput( specMgr->StrDictToSpec( dict, spec ) );
    } else {
        if( P4GODB_CALLS )
            fprintf( stderr, "[P4] OutputStat() - Passing StrDict\n" );
        StrBufDict* ndict = new StrBufDict();
        StrDictIterator* iter = dict->GetIterator();
        StrRef var, val;
        while( iter->Get( var, val ) ) {
            iter->Next();
            if( var == "specdef" || var == "func" || var == "specFormatted" )
                continue;
            ndict->SetVar( var, val );
        }
        ProcessOutput( ndict );
    }
}

/*
 * Diff support for Go API. Since the Diff class only writes its output
 * to files, we run the requested diff putting the output into a temporary
 * file. Then we read the file in and add its contents line by line to the
 * results.
 */

void
P4GoClientUser::Diff( FileSys* f1,
                      FileSys* f2,
                      int doPage,
                      char* diffFlags,
                      Error* e )
{
    if( P4GODB_CALLS )
        fprintf( stderr, "[P4] Diff() - comparing files\n" );

    //
    // Duck binary files. Much the same as ClientUser::Diff, we just
    // put the output into Go space rather than stdout.
    //
    if( !f1->IsTextual() || !f2->IsTextual() ) {
        if( f1->Compare( f2, e ) )
            results.AddOutput( StrRef( "(... files differ ...)" ) );
        return;
    }

    // Time to diff the two text files. Need to ensure that the
    // files are in binary mode, so we have to create new FileSys
    // objects to do this.

    FileSys* f1_bin = FileSys::Create( FST_BINARY );
    FileSys* f2_bin = FileSys::Create( FST_BINARY );
    FileSys* t = FileSys::CreateGlobalTemp( f1->GetType() );

    f1_bin->Set( f1->Name() );
    f2_bin->Set( f2->Name() );

    {
        //
        // In its own block to make sure that the diff object is deleted
        // before we delete the FileSys objects.
        //
#ifndef OS_NEXT
        ::
#endif
          Diff d;

        d.SetInput( f1_bin, f2_bin, diffFlags, e );
        if( !e->Test() )
            d.SetOutput( t->Name(), e );
        if( !e->Test() )
            d.DiffWithFlags( diffFlags );
        d.CloseOutput( e );

        // OK, now we have the diff output, read it in and add it to
        // the output.
        if( !e->Test() )
            t->Open( FOM_READ, e );
        if( !e->Test() ) {
            StrBuf b;
            while( t->ReadLine( &b, e ) )
                results.AddOutput( StrRef( b.Text(), b.Length() ) );
        }
    }

    delete t;
    delete f1_bin;
    delete f2_bin;

    if( e->Test() )
        HandleError( e );
}

/*
 * Return the FileSys to use for a streamed file transfer (e.g. 'p4 print').
 * When a FileSysBuffer has been registered via SetFileBuffer(), the transfer
 * is streamed straight to Go instead of going through a temp file.
 */

FileSys*
P4GoClientUser::File( FileSysType type )
{
    if( P4GODB_CALLS )
        fprintf( stderr, "[P4] File()\n" );

    return FileSys::Create( type, fileBuffer );
}

/*
 * convert input from the user into a form digestible to Perforce. This
 * involves either (a) converting any supplied hash to a Perforce form, or
 * (b) running to_s on whatever we were given.
 */

void
P4GoClientUser::InputData( StrBuf* strbuf, Error* e )
{
    if( P4GODB_CALLS )
        fprintf( stderr, "[P4] InputData(). Using supplied input\n" );

    strbuf->Clear();
    if( input->Count() ) {
        *strbuf = *input->Get( 0 );
        input->Remove( 0 );
    }
}

/*
 * In a script we don't really want the user to see a prompt, so we
 * (ab)use the SetInput() function to allow the caller to supply the
 * answer before the question is asked.
 */

void
P4GoClientUser::Prompt( const StrPtr& msg, StrBuf& rsp, int noEcho, Error* e )
{
    if( P4GODB_CALLS )
        fprintf( stderr, "[P4] Prompt(): %s\n", msg.Text() );

    InputData( &rsp, e );
}

// Unique codes (Subsystem()<<10 | SubCode(), the same fields
// Error::CheckIdi() compares) for p4/msgs/msgserver.h's OldPasswordPrompt
// and NewPasswordPrompt ErrorIds. That header isn't part of the public
// P4API SDK this builds against, so the codes are reconstructed here;
// ES_SERVER is 7.
static const int OldPasswordPromptCode = ( 7 << 10 ) | 906;
static const int NewPasswordPromptCode = ( 7 << 10 ) | 907;

/*
 * 'p4 password' skips the old-password prompt when the caller already
 * holds a valid ticket, or when no password is set yet. RunPassword always
 * queues an old-password answer regardless (see SetDiscardUnusedOldPasswordSlot),
 * so when it does, and the prompt is skipped, discard that unused answer
 * here rather than let it be consumed by the new-password prompt instead.
 * Gated behind discardUnusedOldPasswordSlot so any other caller that hands
 * this command its own hand-counted SetInput() answers (e.g. a superuser
 * resetting someone else's password, which is never asked for an old one)
 * sees no change in behaviour.
 */

void
P4GoClientUser::Prompt( Error* err, StrBuf& rsp, int noEcho, Error* e )
{
    ErrorId* id = err->GetId( 0 );
    int code = id ? id->UniqueCode() : 0;

    if( code == OldPasswordPromptCode )
        oldPasswordPromptSeen = true;
    else if( code == NewPasswordPromptCode )
    {
        if( discardUnusedOldPasswordSlot && !oldPasswordPromptSeen )
        {
            StrBuf discard;
            InputData( &discard, e );
        }
        discardUnusedOldPasswordSlot = false;
    }

    ClientUser::Prompt( err, rsp, noEcho, e );
}

/*
 * Do a resolve. We implement a resolve by calling a block.
 */

int
P4GoClientUser::Resolve( ClientMerge* m, Error* e )
{
    if( P4GODB_CALLS )
        fprintf( stderr, "[P4] Resolve()\n" );

    //
    // If no handler has been set, default to using the merger's resolve
    //
    if( !resolveHandler )
        return m->Resolve( e );

    P4GoMergeData md( this, m, 0 );
    return resolveHandler->Resolve( &md );
}

int
P4GoClientUser::Resolve( ClientResolveA* m, int preview, Error* e )
{
    if( P4GODB_CALLS )
        fprintf( stderr, "[P4] Resolve(Action)\n" );

    //
    // If no resolveHandler has been set, default to using the merger's resolve
    //
    if( !resolveHandler )
        return m->Resolve( 0, e );

    P4GoMergeData md( this, m, 0 );
    return resolveHandler->Resolve( &md );
}

/*
 * Return the ClientProgress.
 */

ClientProgress*
P4GoClientUser::CreateProgress( int type )
{
    if( P4GODB_CALLS )
        fprintf( stderr, "[P4] CreateProgress()\n" );

    if( progress ) {
        return new P4GoClientProgress( progress, type );
    }
    return 0;
}

/*
 * Simple method to check if a progress indicator has been
 * registered to this ClientUser.
 */

int
P4GoClientUser::ProgressIndicator()
{
    if( P4GODB_CALLS )
        fprintf( stderr, "[P4] ProgressIndicator()\n" );
    return progress != NULL;
}

/*
 * Accept input from Go and convert to a StrBuf for Perforce
 * purposes.  We just save what we're given here because we may not
 * have the specdef available to parse it with at this time.
 */

void
P4GoClientUser::ResetInput()
{
    if( P4GODB_CALLS )
        fprintf( stderr, "[P4] ResetInput()\n" );

    input->Clear();
}

void
P4GoClientUser::AppendInput( char* i )
{
    if( P4GODB_CALLS )
        fprintf( stderr, "[P4] AppendInput()\n" );

    input->Put()->Set( i );
}

void
P4GoClientUser::AppendInputBytes( const char* i, unsigned int length )
{
    if( P4GODB_CALLS )
        fprintf( stderr, "[P4] AppendInputBytes()\n" );

    StrBuf* value = input->Put();
    if( length )
        value->Set( i, length );
    else
        value->Clear();
}

/*
 * Set the Handler object. Double-check that it is either nil or
 * an instance of OutputHandler to avoid future problems
 */

void
P4GoClientUser::SetHandler( P4GoHandler* h )
{
    if( P4GODB_CALLS )
        fprintf( stderr, "[P4] SetHandler()\n" );
    handler = h;
    alive = 1; // ensure that we don't drop out after the next call
}

/*
 * Set a ClientProgress for the current ClientUser.
 */
void
P4GoClientUser::SetProgress( P4GoProgress* p )
{
    if( P4GODB_CALLS )
        fprintf( stderr, "[P4] SetProgress()\n" );

    progress = p;
    alive = 1;
}

void
P4GoClientUser::SetSSOHandler( P4GoSSOHandler* h )
{
    if( P4GODB_CALLS )
        fprintf( stderr, "[P4] SetSSOHandler()\n" );
    ClientUser::SetSSOHandler( h );
    alive = 1; // ensure that we don't drop out after the next call
}

P4GoSSOHandler*
P4GoClientUser::GetSSOHandler()
{
    if( P4GODB_CALLS )
        fprintf( stderr, "[P4] GetSSOHandler()\n" );
    return (P4GoSSOHandler*)ClientUser::GetSSOHandler();
}

void
P4GoClientUser::SetResolveHandler( P4GoResolveHandler* h )
{
    if( P4GODB_CALLS )
        fprintf( stderr, "[P4] SetResolveHandler()\n" );
    resolveHandler = h;
}

P4GoResolveHandler*
P4GoClientUser::GetResolveHandler()
{
    if( P4GODB_CALLS )
        fprintf( stderr, "[P4] GetResolveHandler()\n" );
    return resolveHandler;
}

void
P4GoClientUser::SetFileBuffer( P4GoFileSysBuffer* b )
{
    if( P4GODB_CALLS )
        fprintf( stderr, "[P4] SetFileBuffer()\n" );
    fileBuffer = b;
}
