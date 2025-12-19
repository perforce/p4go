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

typedef struct P4GoClientApi P4GoClientApi;
typedef struct Error Error;
typedef struct P4GoResult P4GoResult;
typedef struct P4GoProgress P4GoProgress;
typedef struct StrDict StrDict;
typedef struct StrBufDict StrBufDict;
typedef struct P4GoHandler P4GoHandler;
typedef struct P4GoSSOHandler P4GoSSOHandler;
typedef struct P4GoResolveHandler P4GoResolveHandler;
typedef struct P4GoSpecData P4GoSpecData;
typedef struct MapApi MapApi;
typedef struct P4GoMergeData P4GoMergeData;

#ifdef __cplusplus
extern "C"
{
#endif

    P4GoClientApi* NewClientApi();
    void FreeClientApi( P4GoClientApi* api );

    // Getters and Setters
    const char* P4Identify( P4GoClientApi* api );

    // Control Methods
    int P4Connect( P4GoClientApi* api, Error* e );
    int P4Connected( P4GoClientApi* api );
    int P4Disconnect( P4GoClientApi* api, Error* e );
    void Run( P4GoClientApi* api, char* cmd, int argc, char** argv, Error* e );

    // Result handlers
    int ResultCount( P4GoClientApi* api );
    int ResultGet( P4GoClientApi* api, int index, int* type, P4GoResult** ret );
    const char* ResultGetString( P4GoResult* ret );
    const char* ResultGetBinary( P4GoResult* ret, int* len );
    Error* ResultGetError( P4GoResult* ret );
    P4GoSpecData* ResultGetSpec( P4GoResult* ret );
    int ResultGetKeyPair( P4GoResult* ret, int index, char** var, char** val );

    int IsIgnored( P4GoClientApi* api, char* path );

    //
    // Getters and Setters
    //

    int GetApiLevel( P4GoClientApi* api );
    void SetApiLevel( P4GoClientApi* api, int apiLevel );
    int GetStreams( P4GoClientApi* api );
    void SetStreams( P4GoClientApi* api, int enableStreams );
    int GetTagged( P4GoClientApi* api );
    void SetTagged( P4GoClientApi* api, int enableTagged );
    int GetTrack( P4GoClientApi* api );
    int SetTrack( P4GoClientApi* api, int enableTrack, Error* e );
    int GetGraph( P4GoClientApi* api );
    void SetGraph( P4GoClientApi* api, int enableGraph );
    int GetDebug( P4GoClientApi* api );
    void SetDebug( P4GoClientApi* api, int debug );
    const char* GetCharset( P4GoClientApi* api );
    int SetCharset( P4GoClientApi* api, char* charset, Error* e );
    const char* GetCwd( P4GoClientApi* api );
    void SetCwd( P4GoClientApi* api, char* cwd );
    const char* GetClient( P4GoClientApi* api );
    void SetClient( P4GoClientApi* api, char* client );
    const char* GetEnv( P4GoClientApi* api, char* env );
    int SetEnv( P4GoClientApi* api, char* env, char* value, Error* e );
    const char* GetEnviroFile( P4GoClientApi* api );
    void SetEnviroFile( P4GoClientApi* api, char* enviroFile );
    const char* GetEVar( P4GoClientApi* api, char* evar );
    void SetEVar( P4GoClientApi* api, char* evar, char* value );
    const char* GetHost( P4GoClientApi* api );
    void SetHost( P4GoClientApi* api, char* host );
    const char* GetIgnoreFile( P4GoClientApi* api );
    void SetIgnoreFile( P4GoClientApi* api, char* ignoreFile );
    const char* GetLanguage( P4GoClientApi* api );
    void SetLanguage( P4GoClientApi* api, char* language );
    const char* GetP4ConfigFile( P4GoClientApi* api );
    const char* GetPassword( P4GoClientApi* api );
    void SetPassword( P4GoClientApi* api, char* password );
    const char* GetPort( P4GoClientApi* api );
    void SetPort( P4GoClientApi* api, char* port );
    const char* GetProg( P4GoClientApi* api );
    void SetProg( P4GoClientApi* api, char* prog );
    void SetProtocol( P4GoClientApi* api, char* protocol, char* value );
    void SetVar( P4GoClientApi* api, char* variable, char* value );
    const char* GetTicketFile( P4GoClientApi* api );
    void SetTicketFile( P4GoClientApi* api, char* ticketFile );
    const char* GetTrustFile( P4GoClientApi* api );
    void SetTrustFile( P4GoClientApi* api, char* trustFile );
    const char* GetUser( P4GoClientApi* api );
    void SetUser( P4GoClientApi* api, char* user );
    const char* GetP4Version( P4GoClientApi* api );
    void SetP4Version( P4GoClientApi* api, char* version );
    int GetMaxResults( P4GoClientApi* api );
    void SetMaxResults( P4GoClientApi* api, int maxResults );
    int GetMaxScanRows( P4GoClientApi* api );
    void SetMaxScanRows( P4GoClientApi* api, int maxScanRows );
    int GetMaxLockTime( P4GoClientApi* api );
    void SetMaxLockTime( P4GoClientApi* api, int maxLockTime );

    void ResetInput( P4GoClientApi* api );
    void AppendInput( P4GoClientApi* api, char* input );

    P4GoSpecData* ParseSpec( P4GoClientApi* api, char* spec, char* form, Error* e );
    char* FormatSpec( P4GoClientApi* api, char* spec, StrDict* dict, Error* e );

    int P4ServerLevel( P4GoClientApi* api, Error* e );
    int P4ServerCaseSensitive( P4GoClientApi* api, Error* e );
    int P4ServerUnicode( P4GoClientApi* api, Error* e );

    //
    // Error wrapper
    //
    Error* MakeError();
    void FreeError( Error* e );
    const char* FmtError( Error* e, int i );
    int GetErrorCode( Error* e, int i );
    int GetErrorCount( Error* e );
    int GetErrorSeverity( Error* e );
    int GetErrorSeverityI( Error* e, int i );
    StrDict* GetDict( Error* e );

    // Callbacks

    P4GoProgress* NewProgress();
    void FreeProgress( P4GoProgress* progress );
    void SetProgress( P4GoClientApi* api, P4GoProgress* progress );
    P4GoProgress* GetProgress( P4GoClientApi* api );

    P4GoHandler* NewHandler();
    void FreeHandler( P4GoHandler* handler );
    void SetHandler( P4GoClientApi* api, P4GoHandler* handler );
    P4GoHandler* GetHandler( P4GoClientApi* api );

    P4GoSSOHandler* NewSSOHandler();
    void FreeSSOHandler( P4GoSSOHandler* handler );
    void SetSSOHandler( P4GoClientApi* api, P4GoSSOHandler* handler );
    P4GoSSOHandler* GetSSOHandler( P4GoClientApi* api );

    int StrDictGetKeyPair( StrDict* dict, int index, char** var, char** val );
    void FreeStrDict( StrDict* dict );
    void StrDictSetKeyPair( StrDict* dict, char* var, char* val );
    StrDict* NewStrDict();
    void FreeStrDict( StrDict* dict );

    int SpecDataGetKeyPair( P4GoSpecData* spec,
                            int index,
                            char** var,
                            char** val );
    void FreeSpecData( P4GoSpecData* spec );

    void SetResolveHandler( P4GoClientApi* api, P4GoResolveHandler* handler );
    P4GoResolveHandler* GetResolveHandler( P4GoClientApi* api );
    P4GoResolveHandler* NewResolveHandler();
    void FreeResolveHandler( P4GoResolveHandler* handler );


    char* MergeDataGetYourName( P4GoMergeData* m );
    char* MergeDataGetTheirName( P4GoMergeData* m );
    char* MergeDataGetBaseName( P4GoMergeData* m );

    char* MergeDataGetYourPath( P4GoMergeData* m );
    char* MergeDataGetTheirPath( P4GoMergeData* m );
    char* MergeDataGetBasePath( P4GoMergeData* m );
    char* MergeDataGetResultPath( P4GoMergeData* m );

    int MergeDataRunMergeTool( P4GoMergeData* m );

    int MergeDataGetActionResolveStatus( P4GoMergeData* m );
    int MergeDataGetContentResolveStatus( P4GoMergeData* m );

    void* MergeDataGetMergeInfo( P4GoMergeData* m );

    const Error* MergeDataGetMergeAction( P4GoMergeData* m );
    const Error* MergeDataGetYoursAction( P4GoMergeData* m );
    const Error* MergeDataGetTheirAction( P4GoMergeData* m );
    const Error* MergeDataGetType( P4GoMergeData* m );

    char* MergeDataGetString( P4GoMergeData* m );
    int MergeDataGetMergeHint( P4GoMergeData* m );

    // MapApi

    MapApi* NewMapApi();
    void FreeMapApi( MapApi* mapapi );
    MapApi* JoinMapApi( MapApi* m1, MapApi* m2 );
    void MapApiInsert( MapApi* mapapi, char* lhs, char* rhs, int flag );
    void MapApiClear( MapApi* mapapi );
    int MapApiCount( MapApi* mapapi );
    MapApi* MapApiReverse( MapApi* mapapi );
    char* MapApiLhs( MapApi* mapapi, int i );
    char* MapApiRhs( MapApi* mapapi, int i );
    int MapApiType( MapApi* mapapi, int i );
    char* MapApiTranslate( MapApi* mapapi, char* input, int dir );
    char** MapApiTranslateArray( MapApi* mapapi,
                                 char* input,
                                 int dir,
                                 int* results );

    // Spec data access functions
    int SpecDataGetVar(P4GoSpecData* sd, const char* key, char** value);
    int SpecDataGetVarCount(P4GoSpecData* sd);
    int SpecDataGetKey(P4GoSpecData* sd, int index, char** key);
    int SpecDataIsArray(P4GoSpecData* sd, const char* key);
    int SpecDataGetArraySize(P4GoSpecData* sd, const char* key);
    int SpecDataGetArrayValue(P4GoSpecData* sd, const char* key, int index, char** value);
    void FreeSpecData(P4GoSpecData* sd);

#ifdef __cplusplus
}
#endif