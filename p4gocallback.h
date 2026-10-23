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

void
cbProgressInit( void* progress, int type );
void
cbProgressDescription( void* progress, char* d, int u );
void
cbProgressTotal( void* progress, long total );
void
cbProgressUpdate( void* progress, long position );
void
cbProgressDone( void* progress, int fail );

int
cbIsAlive( void* keepAlive );

int
cbHandleBinary( void* handler, char* d, int len );
int
cbHandleMessage( void* handler, Error* e );
int
cbHandleStat( void* handler, StrDict* d );
int
cbHandleSpec( void* handler, P4GoSpecData* d );
int
cbHandleText( void* handler, char* d );
int
cbHandleTrack( void* handler, char* d );

int
cbSSOAuthorize( void* handler, StrDict* d, int l, char** r );

int
cbResolve( void* handler, P4GoMergeData* m );

void
cbFbRelease( void* handler );
void
cbFbOpen( void* handler, char *path, int mode, Error *e );
void
cbFbWrite( void* handler, char *buf, int len, Error *e );
int
cbFbRead( void* handler, char *buf, int len, Error *e );
void
cbFbClose( void* handler, Error *e );
void
cbFbSeek( void* handler, long long offset, Error* e );
void
cbFbSizeHint( void* handler, long long size );