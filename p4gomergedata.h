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

class P4GoMergeData
{
  public:
    P4GoMergeData( ClientUser* ui, ClientMerge* m, void* info );
    P4GoMergeData( ClientUser* ui, ClientResolveA* m, void* info );

    void SetDebug( int d ) { debug = d; }

    //	Content resolve
    char* GetYourName();
    char* GetTheirName();
    char* GetBaseName();

    char* GetYourPath();
    char* GetTheirPath();
    char* GetBasePath();
    char* GetResultPath();

    int RunMergeTool();

    //	What type of resolve is it?
    int GetActionResolveStatus();
    int GetContentResolveStatus();

    //	Action Resolve
    void* GetMergeInfo();

    const Error* GetMergeAction();
    const Error* GetYoursAction();
    const Error* GetTheirAction();
    const Error* GetType();


    char* GetString();
    int GetMergeHint();

  private:
    int debug;
    ClientUser* ui;
    MergeStatus hint;
    ClientMerge* merger;
    ClientResolveA* actionmerger;
    StrBuf yours;
    StrBuf theirs;
    StrBuf base;
    StrBuf asStr;

    void* info;
};
