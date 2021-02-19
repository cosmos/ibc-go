<!DOCTYPE html>

<html>
  <head>
    <title>Protocol Documentation</title>
    <meta charset="UTF-8">
    <link rel="stylesheet" type="text/css" href="https://fonts.googleapis.com/css?family=Ubuntu:400,700,400italic"/>
    <style>
      body {
        width: 60em;
        margin: 1em auto;
        color: #222;
        font-family: "Ubuntu", sans-serif;
        padding-bottom: 4em;
      }

      h1 {
        font-weight: normal;
        border-bottom: 1px solid #aaa;
        padding-bottom: 0.5ex;
      }

      h2 {
        border-bottom: 1px solid #aaa;
        padding-bottom: 0.5ex;
        margin: 1.5em 0;
      }

      h3 {
        font-weight: normal;
        border-bottom: 1px solid #aaa;
        padding-bottom: 0.5ex;
      }

      a {
        text-decoration: none;
        color: #567e25;
      }

      table {
        width: 100%;
        font-size: 80%;
        border-collapse: collapse;
      }

      thead {
        font-weight: 700;
        background-color: #dcdcdc;
      }

      tbody tr:nth-child(even) {
        background-color: #fbfbfb;
      }

      td {
        border: 1px solid #ccc;
        padding: 0.5ex 2ex;
      }

      td p {
        text-indent: 1em;
        margin: 0;
      }

      td p:nth-child(1) {
        text-indent: 0;  
      }

       
      .field-table td:nth-child(1) {  
        width: 10em;
      }
      .field-table td:nth-child(2) {  
        width: 10em;
      }
      .field-table td:nth-child(3) {  
        width: 6em;
      }
      .field-table td:nth-child(4) {  
        width: auto;
      }

       
      .extension-table td:nth-child(1) {  
        width: 10em;
      }
      .extension-table td:nth-child(2) {  
        width: 10em;
      }
      .extension-table td:nth-child(3) {  
        width: 10em;
      }
      .extension-table td:nth-child(4) {  
        width: 5em;
      }
      .extension-table td:nth-child(5) {  
        width: auto;
      }

       
      .enum-table td:nth-child(1) {  
        width: 10em;
      }
      .enum-table td:nth-child(2) {  
        width: 10em;
      }
      .enum-table td:nth-child(3) {  
        width: auto;
      }

       
      .scalar-value-types-table tr {
        height: 3em;
      }

       
      #toc-container ul {
        list-style-type: none;
        padding-left: 1em;
        line-height: 180%;
        margin: 0;
      }
      #toc > li > a {
        font-weight: bold;
      }

       
      .file-heading {
        width: 100%;
        display: table;
        border-bottom: 1px solid #aaa;
        margin: 4em 0 1.5em 0;
      }
      .file-heading h2 {
        border: none;
        display: table-cell;
      }
      .file-heading a {
        text-align: right;
        display: table-cell;
      }

       
      .badge {
        width: 1.6em;
        height: 1.6em;
        display: inline-block;

        line-height: 1.6em;
        text-align: center;
        font-weight: bold;
        font-size: 60%;

        color: #89ba48;
        background-color: #dff0c8;

        margin: 0.5ex 1em 0.5ex -1em;
        border: 1px solid #fbfbfb;
        border-radius: 1ex;
      }
    </style>

    
    <link rel="stylesheet" type="text/css" href="stylesheet.css"/>
  </head>

  <body>

    <h1 id="title">Protocol Documentation</h1>

    <h2>Table of Contents</h2>

    <div id="toc-container">
      <ul id="toc">
        
          
          <li>
            <a href="#ibcgo%2fcore%2fclient%2fv1%2fclient.proto">ibcgo/core/client/v1/client.proto</a>
            <ul>
              
                <li>
                  <a href="#ibcgo.core.client.v1.ClientConsensusStates"><span class="badge">M</span>ClientConsensusStates</a>
                </li>
              
                <li>
                  <a href="#ibcgo.core.client.v1.ClientUpdateProposal"><span class="badge">M</span>ClientUpdateProposal</a>
                </li>
              
                <li>
                  <a href="#ibcgo.core.client.v1.ConsensusStateWithHeight"><span class="badge">M</span>ConsensusStateWithHeight</a>
                </li>
              
                <li>
                  <a href="#ibcgo.core.client.v1.Height"><span class="badge">M</span>Height</a>
                </li>
              
                <li>
                  <a href="#ibcgo.core.client.v1.IdentifiedClientState"><span class="badge">M</span>IdentifiedClientState</a>
                </li>
              
                <li>
                  <a href="#ibcgo.core.client.v1.Params"><span class="badge">M</span>Params</a>
                </li>
              
              
              
              
            </ul>
          </li>
        
          
          <li>
            <a href="#ibcgo%2fcore%2fclient%2fv1%2fgenesis.proto">ibcgo/core/client/v1/genesis.proto</a>
            <ul>
              
                <li>
                  <a href="#ibcgo.core.client.v1.GenesisMetadata"><span class="badge">M</span>GenesisMetadata</a>
                </li>
              
                <li>
                  <a href="#ibcgo.core.client.v1.GenesisState"><span class="badge">M</span>GenesisState</a>
                </li>
              
                <li>
                  <a href="#ibcgo.core.client.v1.IdentifiedGenesisMetadata"><span class="badge">M</span>IdentifiedGenesisMetadata</a>
                </li>
              
              
              
              
            </ul>
          </li>
        
          
          <li>
            <a href="#ibcgo%2fcore%2fclient%2fv1%2fquery.proto">ibcgo/core/client/v1/query.proto</a>
            <ul>
              
                <li>
                  <a href="#ibcgo.core.client.v1.QueryClientParamsRequest"><span class="badge">M</span>QueryClientParamsRequest</a>
                </li>
              
                <li>
                  <a href="#ibcgo.core.client.v1.QueryClientParamsResponse"><span class="badge">M</span>QueryClientParamsResponse</a>
                </li>
              
                <li>
                  <a href="#ibcgo.core.client.v1.QueryClientStateRequest"><span class="badge">M</span>QueryClientStateRequest</a>
                </li>
              
                <li>
                  <a href="#ibcgo.core.client.v1.QueryClientStateResponse"><span class="badge">M</span>QueryClientStateResponse</a>
                </li>
              
                <li>
                  <a href="#ibcgo.core.client.v1.QueryClientStatesRequest"><span class="badge">M</span>QueryClientStatesRequest</a>
                </li>
              
                <li>
                  <a href="#ibcgo.core.client.v1.QueryClientStatesResponse"><span class="badge">M</span>QueryClientStatesResponse</a>
                </li>
              
                <li>
                  <a href="#ibcgo.core.client.v1.QueryConsensusStateRequest"><span class="badge">M</span>QueryConsensusStateRequest</a>
                </li>
              
                <li>
                  <a href="#ibcgo.core.client.v1.QueryConsensusStateResponse"><span class="badge">M</span>QueryConsensusStateResponse</a>
                </li>
              
                <li>
                  <a href="#ibcgo.core.client.v1.QueryConsensusStatesRequest"><span class="badge">M</span>QueryConsensusStatesRequest</a>
                </li>
              
                <li>
                  <a href="#ibcgo.core.client.v1.QueryConsensusStatesResponse"><span class="badge">M</span>QueryConsensusStatesResponse</a>
                </li>
              
              
              
              
                <li>
                  <a href="#ibcgo.core.client.v1.Query"><span class="badge">S</span>Query</a>
                </li>
              
            </ul>
          </li>
        
          
          <li>
            <a href="#ibcgo%2fcore%2fclient%2fv1%2ftx.proto">ibcgo/core/client/v1/tx.proto</a>
            <ul>
              
                <li>
                  <a href="#ibcgo.core.client.v1.MsgCreateClient"><span class="badge">M</span>MsgCreateClient</a>
                </li>
              
                <li>
                  <a href="#ibcgo.core.client.v1.MsgCreateClientResponse"><span class="badge">M</span>MsgCreateClientResponse</a>
                </li>
              
                <li>
                  <a href="#ibcgo.core.client.v1.MsgSubmitMisbehaviour"><span class="badge">M</span>MsgSubmitMisbehaviour</a>
                </li>
              
                <li>
                  <a href="#ibcgo.core.client.v1.MsgSubmitMisbehaviourResponse"><span class="badge">M</span>MsgSubmitMisbehaviourResponse</a>
                </li>
              
                <li>
                  <a href="#ibcgo.core.client.v1.MsgUpdateClient"><span class="badge">M</span>MsgUpdateClient</a>
                </li>
              
                <li>
                  <a href="#ibcgo.core.client.v1.MsgUpdateClientResponse"><span class="badge">M</span>MsgUpdateClientResponse</a>
                </li>
              
                <li>
                  <a href="#ibcgo.core.client.v1.MsgUpgradeClient"><span class="badge">M</span>MsgUpgradeClient</a>
                </li>
              
                <li>
                  <a href="#ibcgo.core.client.v1.MsgUpgradeClientResponse"><span class="badge">M</span>MsgUpgradeClientResponse</a>
                </li>
              
              
              
              
                <li>
                  <a href="#ibcgo.core.client.v1.Msg"><span class="badge">S</span>Msg</a>
                </li>
              
            </ul>
          </li>
        
        <li><a href="#scalar-value-types">Scalar Value Types</a></li>
      </ul>
    </div>

    
      
      <div class="file-heading">
        <h2 id="ibcgo/core/client/v1/client.proto">ibcgo/core/client/v1/client.proto</h2><a href="#title">Top</a>
      </div>
      <p></p>

      
        <h3 id="ibcgo.core.client.v1.ClientConsensusStates">ClientConsensusStates</h3>
        <p>ClientConsensusStates defines all the stored consensus states for a given</p><p>client.</p>

        
          <table class="field-table">
            <thead>
              <tr><td>Field</td><td>Type</td><td>Label</td><td>Description</td></tr>
            </thead>
            <tbody>
              
                <tr>
                  <td>client_id</td>
                  <td><a href="#string">string</a></td>
                  <td></td>
                  <td><p>client identifier </p></td>
                </tr>
              
                <tr>
                  <td>consensus_states</td>
                  <td><a href="#ibcgo.core.client.v1.ConsensusStateWithHeight">ConsensusStateWithHeight</a></td>
                  <td>repeated</td>
                  <td><p>consensus states and their heights associated with the client </p></td>
                </tr>
              
            </tbody>
          </table>

          

        
      
        <h3 id="ibcgo.core.client.v1.ClientUpdateProposal">ClientUpdateProposal</h3>
        <p>ClientUpdateProposal is a governance proposal. If it passes, the substitute</p><p>client's consensus states starting from the 'initial height' are copied over</p><p>to the subjects client state. The proposal handler may fail if the subject</p><p>and the substitute do not match in client and chain parameters (with</p><p>exception to latest height, frozen height, and chain-id). The updated client</p><p>must also be valid (cannot be expired).</p>

        
          <table class="field-table">
            <thead>
              <tr><td>Field</td><td>Type</td><td>Label</td><td>Description</td></tr>
            </thead>
            <tbody>
              
                <tr>
                  <td>title</td>
                  <td><a href="#string">string</a></td>
                  <td></td>
                  <td><p>the title of the update proposal </p></td>
                </tr>
              
                <tr>
                  <td>description</td>
                  <td><a href="#string">string</a></td>
                  <td></td>
                  <td><p>the description of the proposal </p></td>
                </tr>
              
                <tr>
                  <td>subject_client_id</td>
                  <td><a href="#string">string</a></td>
                  <td></td>
                  <td><p>the client identifier for the client to be updated if the proposal passes </p></td>
                </tr>
              
                <tr>
                  <td>substitute_client_id</td>
                  <td><a href="#string">string</a></td>
                  <td></td>
                  <td><p>the substitute client identifier for the client standing in for the subject
client </p></td>
                </tr>
              
                <tr>
                  <td>initial_height</td>
                  <td><a href="#ibcgo.core.client.v1.Height">Height</a></td>
                  <td></td>
                  <td><p>the intital height to copy consensus states from the substitute to the
subject </p></td>
                </tr>
              
            </tbody>
          </table>

          

        
      
        <h3 id="ibcgo.core.client.v1.ConsensusStateWithHeight">ConsensusStateWithHeight</h3>
        <p>ConsensusStateWithHeight defines a consensus state with an additional height</p><p>field.</p>

        
          <table class="field-table">
            <thead>
              <tr><td>Field</td><td>Type</td><td>Label</td><td>Description</td></tr>
            </thead>
            <tbody>
              
                <tr>
                  <td>height</td>
                  <td><a href="#ibcgo.core.client.v1.Height">Height</a></td>
                  <td></td>
                  <td><p>consensus state height </p></td>
                </tr>
              
                <tr>
                  <td>consensus_state</td>
                  <td><a href="#google.protobuf.Any">google.protobuf.Any</a></td>
                  <td></td>
                  <td><p>consensus state </p></td>
                </tr>
              
            </tbody>
          </table>

          

        
      
        <h3 id="ibcgo.core.client.v1.Height">Height</h3>
        <p>Height is a monotonically increasing data type</p><p>that can be compared against another Height for the purposes of updating and</p><p>freezing clients</p><p>Normally the RevisionHeight is incremented at each height while keeping</p><p>RevisionNumber the same. However some consensus algorithms may choose to</p><p>reset the height in certain conditions e.g. hard forks, state-machine</p><p>breaking changes In these cases, the RevisionNumber is incremented so that</p><p>height continues to be monitonically increasing even as the RevisionHeight</p><p>gets reset</p>

        
          <table class="field-table">
            <thead>
              <tr><td>Field</td><td>Type</td><td>Label</td><td>Description</td></tr>
            </thead>
            <tbody>
              
                <tr>
                  <td>revision_number</td>
                  <td><a href="#uint64">uint64</a></td>
                  <td></td>
                  <td><p>the revision that the client is currently on </p></td>
                </tr>
              
                <tr>
                  <td>revision_height</td>
                  <td><a href="#uint64">uint64</a></td>
                  <td></td>
                  <td><p>the height within the given revision </p></td>
                </tr>
              
            </tbody>
          </table>

          

        
      
        <h3 id="ibcgo.core.client.v1.IdentifiedClientState">IdentifiedClientState</h3>
        <p>IdentifiedClientState defines a client state with an additional client</p><p>identifier field.</p>

        
          <table class="field-table">
            <thead>
              <tr><td>Field</td><td>Type</td><td>Label</td><td>Description</td></tr>
            </thead>
            <tbody>
              
                <tr>
                  <td>client_id</td>
                  <td><a href="#string">string</a></td>
                  <td></td>
                  <td><p>client identifier </p></td>
                </tr>
              
                <tr>
                  <td>client_state</td>
                  <td><a href="#google.protobuf.Any">google.protobuf.Any</a></td>
                  <td></td>
                  <td><p>client state </p></td>
                </tr>
              
            </tbody>
          </table>

          

        
      
        <h3 id="ibcgo.core.client.v1.Params">Params</h3>
        <p>Params defines the set of IBC light client parameters.</p>

        
          <table class="field-table">
            <thead>
              <tr><td>Field</td><td>Type</td><td>Label</td><td>Description</td></tr>
            </thead>
            <tbody>
              
                <tr>
                  <td>allowed_clients</td>
                  <td><a href="#string">string</a></td>
                  <td>repeated</td>
                  <td><p>allowed_clients defines the list of allowed client state types. </p></td>
                </tr>
              
            </tbody>
          </table>

          

        
      

      

      

      
    
      
      <div class="file-heading">
        <h2 id="ibcgo/core/client/v1/genesis.proto">ibcgo/core/client/v1/genesis.proto</h2><a href="#title">Top</a>
      </div>
      <p></p>

      
        <h3 id="ibcgo.core.client.v1.GenesisMetadata">GenesisMetadata</h3>
        <p>GenesisMetadata defines the genesis type for metadata that clients may return</p><p>with ExportMetadata</p>

        
          <table class="field-table">
            <thead>
              <tr><td>Field</td><td>Type</td><td>Label</td><td>Description</td></tr>
            </thead>
            <tbody>
              
                <tr>
                  <td>key</td>
                  <td><a href="#bytes">bytes</a></td>
                  <td></td>
                  <td><p>store key of metadata without clientID-prefix </p></td>
                </tr>
              
                <tr>
                  <td>value</td>
                  <td><a href="#bytes">bytes</a></td>
                  <td></td>
                  <td><p>metadata value </p></td>
                </tr>
              
            </tbody>
          </table>

          

        
      
        <h3 id="ibcgo.core.client.v1.GenesisState">GenesisState</h3>
        <p>GenesisState defines the ibc client submodule's genesis state.</p>

        
          <table class="field-table">
            <thead>
              <tr><td>Field</td><td>Type</td><td>Label</td><td>Description</td></tr>
            </thead>
            <tbody>
              
                <tr>
                  <td>clients</td>
                  <td><a href="#ibcgo.core.client.v1.IdentifiedClientState">IdentifiedClientState</a></td>
                  <td>repeated</td>
                  <td><p>client states with their corresponding identifiers </p></td>
                </tr>
              
                <tr>
                  <td>clients_consensus</td>
                  <td><a href="#ibcgo.core.client.v1.ClientConsensusStates">ClientConsensusStates</a></td>
                  <td>repeated</td>
                  <td><p>consensus states from each client </p></td>
                </tr>
              
                <tr>
                  <td>clients_metadata</td>
                  <td><a href="#ibcgo.core.client.v1.IdentifiedGenesisMetadata">IdentifiedGenesisMetadata</a></td>
                  <td>repeated</td>
                  <td><p>metadata from each client </p></td>
                </tr>
              
                <tr>
                  <td>params</td>
                  <td><a href="#ibcgo.core.client.v1.Params">Params</a></td>
                  <td></td>
                  <td><p> </p></td>
                </tr>
              
                <tr>
                  <td>create_localhost</td>
                  <td><a href="#bool">bool</a></td>
                  <td></td>
                  <td><p>create localhost on initialization </p></td>
                </tr>
              
                <tr>
                  <td>next_client_sequence</td>
                  <td><a href="#uint64">uint64</a></td>
                  <td></td>
                  <td><p>the sequence for the next generated client identifier </p></td>
                </tr>
              
            </tbody>
          </table>

          

        
      
        <h3 id="ibcgo.core.client.v1.IdentifiedGenesisMetadata">IdentifiedGenesisMetadata</h3>
        <p>IdentifiedGenesisMetadata has the client metadata with the corresponding</p><p>client id.</p>

        
          <table class="field-table">
            <thead>
              <tr><td>Field</td><td>Type</td><td>Label</td><td>Description</td></tr>
            </thead>
            <tbody>
              
                <tr>
                  <td>client_id</td>
                  <td><a href="#string">string</a></td>
                  <td></td>
                  <td><p> </p></td>
                </tr>
              
                <tr>
                  <td>client_metadata</td>
                  <td><a href="#ibcgo.core.client.v1.GenesisMetadata">GenesisMetadata</a></td>
                  <td>repeated</td>
                  <td><p> </p></td>
                </tr>
              
            </tbody>
          </table>

          

        
      

      

      

      
    
      
      <div class="file-heading">
        <h2 id="ibcgo/core/client/v1/query.proto">ibcgo/core/client/v1/query.proto</h2><a href="#title">Top</a>
      </div>
      <p></p>

      
        <h3 id="ibcgo.core.client.v1.QueryClientParamsRequest">QueryClientParamsRequest</h3>
        <p>QueryClientParamsRequest is the request type for the Query/ClientParams RPC</p><p>method.</p>

        

        
      
        <h3 id="ibcgo.core.client.v1.QueryClientParamsResponse">QueryClientParamsResponse</h3>
        <p>QueryClientParamsResponse is the response type for the Query/ClientParams RPC</p><p>method.</p>

        
          <table class="field-table">
            <thead>
              <tr><td>Field</td><td>Type</td><td>Label</td><td>Description</td></tr>
            </thead>
            <tbody>
              
                <tr>
                  <td>params</td>
                  <td><a href="#ibcgo.core.client.v1.Params">Params</a></td>
                  <td></td>
                  <td><p>params defines the parameters of the module. </p></td>
                </tr>
              
            </tbody>
          </table>

          

        
      
        <h3 id="ibcgo.core.client.v1.QueryClientStateRequest">QueryClientStateRequest</h3>
        <p>QueryClientStateRequest is the request type for the Query/ClientState RPC</p><p>method</p>

        
          <table class="field-table">
            <thead>
              <tr><td>Field</td><td>Type</td><td>Label</td><td>Description</td></tr>
            </thead>
            <tbody>
              
                <tr>
                  <td>client_id</td>
                  <td><a href="#string">string</a></td>
                  <td></td>
                  <td><p>client state unique identifier </p></td>
                </tr>
              
            </tbody>
          </table>

          

        
      
        <h3 id="ibcgo.core.client.v1.QueryClientStateResponse">QueryClientStateResponse</h3>
        <p>QueryClientStateResponse is the response type for the Query/ClientState RPC</p><p>method. Besides the client state, it includes a proof and the height from</p><p>which the proof was retrieved.</p>

        
          <table class="field-table">
            <thead>
              <tr><td>Field</td><td>Type</td><td>Label</td><td>Description</td></tr>
            </thead>
            <tbody>
              
                <tr>
                  <td>client_state</td>
                  <td><a href="#google.protobuf.Any">google.protobuf.Any</a></td>
                  <td></td>
                  <td><p>client state associated with the request identifier </p></td>
                </tr>
              
                <tr>
                  <td>proof</td>
                  <td><a href="#bytes">bytes</a></td>
                  <td></td>
                  <td><p>merkle proof of existence </p></td>
                </tr>
              
                <tr>
                  <td>proof_height</td>
                  <td><a href="#ibcgo.core.client.v1.Height">Height</a></td>
                  <td></td>
                  <td><p>height at which the proof was retrieved </p></td>
                </tr>
              
            </tbody>
          </table>

          

        
      
        <h3 id="ibcgo.core.client.v1.QueryClientStatesRequest">QueryClientStatesRequest</h3>
        <p>QueryClientStatesRequest is the request type for the Query/ClientStates RPC</p><p>method</p>

        
          <table class="field-table">
            <thead>
              <tr><td>Field</td><td>Type</td><td>Label</td><td>Description</td></tr>
            </thead>
            <tbody>
              
                <tr>
                  <td>pagination</td>
                  <td><a href="#cosmos.base.query.v1beta1.PageRequest">cosmos.base.query.v1beta1.PageRequest</a></td>
                  <td></td>
                  <td><p>pagination request </p></td>
                </tr>
              
            </tbody>
          </table>

          

        
      
        <h3 id="ibcgo.core.client.v1.QueryClientStatesResponse">QueryClientStatesResponse</h3>
        <p>QueryClientStatesResponse is the response type for the Query/ClientStates RPC</p><p>method.</p>

        
          <table class="field-table">
            <thead>
              <tr><td>Field</td><td>Type</td><td>Label</td><td>Description</td></tr>
            </thead>
            <tbody>
              
                <tr>
                  <td>client_states</td>
                  <td><a href="#ibcgo.core.client.v1.IdentifiedClientState">IdentifiedClientState</a></td>
                  <td>repeated</td>
                  <td><p>list of stored ClientStates of the chain. </p></td>
                </tr>
              
                <tr>
                  <td>pagination</td>
                  <td><a href="#cosmos.base.query.v1beta1.PageResponse">cosmos.base.query.v1beta1.PageResponse</a></td>
                  <td></td>
                  <td><p>pagination response </p></td>
                </tr>
              
            </tbody>
          </table>

          

        
      
        <h3 id="ibcgo.core.client.v1.QueryConsensusStateRequest">QueryConsensusStateRequest</h3>
        <p>QueryConsensusStateRequest is the request type for the Query/ConsensusState</p><p>RPC method. Besides the consensus state, it includes a proof and the height</p><p>from which the proof was retrieved.</p>

        
          <table class="field-table">
            <thead>
              <tr><td>Field</td><td>Type</td><td>Label</td><td>Description</td></tr>
            </thead>
            <tbody>
              
                <tr>
                  <td>client_id</td>
                  <td><a href="#string">string</a></td>
                  <td></td>
                  <td><p>client identifier </p></td>
                </tr>
              
                <tr>
                  <td>revision_number</td>
                  <td><a href="#uint64">uint64</a></td>
                  <td></td>
                  <td><p>consensus state revision number </p></td>
                </tr>
              
                <tr>
                  <td>revision_height</td>
                  <td><a href="#uint64">uint64</a></td>
                  <td></td>
                  <td><p>consensus state revision height </p></td>
                </tr>
              
                <tr>
                  <td>latest_height</td>
                  <td><a href="#bool">bool</a></td>
                  <td></td>
                  <td><p>latest_height overrrides the height field and queries the latest stored
ConsensusState </p></td>
                </tr>
              
            </tbody>
          </table>

          

        
      
        <h3 id="ibcgo.core.client.v1.QueryConsensusStateResponse">QueryConsensusStateResponse</h3>
        <p>QueryConsensusStateResponse is the response type for the Query/ConsensusState</p><p>RPC method</p>

        
          <table class="field-table">
            <thead>
              <tr><td>Field</td><td>Type</td><td>Label</td><td>Description</td></tr>
            </thead>
            <tbody>
              
                <tr>
                  <td>consensus_state</td>
                  <td><a href="#google.protobuf.Any">google.protobuf.Any</a></td>
                  <td></td>
                  <td><p>consensus state associated with the client identifier at the given height </p></td>
                </tr>
              
                <tr>
                  <td>proof</td>
                  <td><a href="#bytes">bytes</a></td>
                  <td></td>
                  <td><p>merkle proof of existence </p></td>
                </tr>
              
                <tr>
                  <td>proof_height</td>
                  <td><a href="#ibcgo.core.client.v1.Height">Height</a></td>
                  <td></td>
                  <td><p>height at which the proof was retrieved </p></td>
                </tr>
              
            </tbody>
          </table>

          

        
      
        <h3 id="ibcgo.core.client.v1.QueryConsensusStatesRequest">QueryConsensusStatesRequest</h3>
        <p>QueryConsensusStatesRequest is the request type for the Query/ConsensusStates</p><p>RPC method.</p>

        
          <table class="field-table">
            <thead>
              <tr><td>Field</td><td>Type</td><td>Label</td><td>Description</td></tr>
            </thead>
            <tbody>
              
                <tr>
                  <td>client_id</td>
                  <td><a href="#string">string</a></td>
                  <td></td>
                  <td><p>client identifier </p></td>
                </tr>
              
                <tr>
                  <td>pagination</td>
                  <td><a href="#cosmos.base.query.v1beta1.PageRequest">cosmos.base.query.v1beta1.PageRequest</a></td>
                  <td></td>
                  <td><p>pagination request </p></td>
                </tr>
              
            </tbody>
          </table>

          

        
      
        <h3 id="ibcgo.core.client.v1.QueryConsensusStatesResponse">QueryConsensusStatesResponse</h3>
        <p>QueryConsensusStatesResponse is the response type for the</p><p>Query/ConsensusStates RPC method</p>

        
          <table class="field-table">
            <thead>
              <tr><td>Field</td><td>Type</td><td>Label</td><td>Description</td></tr>
            </thead>
            <tbody>
              
                <tr>
                  <td>consensus_states</td>
                  <td><a href="#ibcgo.core.client.v1.ConsensusStateWithHeight">ConsensusStateWithHeight</a></td>
                  <td>repeated</td>
                  <td><p>consensus states associated with the identifier </p></td>
                </tr>
              
                <tr>
                  <td>pagination</td>
                  <td><a href="#cosmos.base.query.v1beta1.PageResponse">cosmos.base.query.v1beta1.PageResponse</a></td>
                  <td></td>
                  <td><p>pagination response </p></td>
                </tr>
              
            </tbody>
          </table>

          

        
      

      

      

      
        <h3 id="ibcgo.core.client.v1.Query">Query</h3>
        <p>Query provides defines the gRPC querier service</p>
        <table class="enum-table">
          <thead>
            <tr><td>Method Name</td><td>Request Type</td><td>Response Type</td><td>Description</td></tr>
          </thead>
          <tbody>
            
              <tr>
                <td>ClientState</td>
                <td><a href="#ibcgo.core.client.v1.QueryClientStateRequest">QueryClientStateRequest</a></td>
                <td><a href="#ibcgo.core.client.v1.QueryClientStateResponse">QueryClientStateResponse</a></td>
                <td><p>ClientState queries an IBC light client.</p></td>
              </tr>
            
              <tr>
                <td>ClientStates</td>
                <td><a href="#ibcgo.core.client.v1.QueryClientStatesRequest">QueryClientStatesRequest</a></td>
                <td><a href="#ibcgo.core.client.v1.QueryClientStatesResponse">QueryClientStatesResponse</a></td>
                <td><p>ClientStates queries all the IBC light clients of a chain.</p></td>
              </tr>
            
              <tr>
                <td>ConsensusState</td>
                <td><a href="#ibcgo.core.client.v1.QueryConsensusStateRequest">QueryConsensusStateRequest</a></td>
                <td><a href="#ibcgo.core.client.v1.QueryConsensusStateResponse">QueryConsensusStateResponse</a></td>
                <td><p>ConsensusState queries a consensus state associated with a client state at
a given height.</p></td>
              </tr>
            
              <tr>
                <td>ConsensusStates</td>
                <td><a href="#ibcgo.core.client.v1.QueryConsensusStatesRequest">QueryConsensusStatesRequest</a></td>
                <td><a href="#ibcgo.core.client.v1.QueryConsensusStatesResponse">QueryConsensusStatesResponse</a></td>
                <td><p>ConsensusStates queries all the consensus state associated with a given
client.</p></td>
              </tr>
            
              <tr>
                <td>ClientParams</td>
                <td><a href="#ibcgo.core.client.v1.QueryClientParamsRequest">QueryClientParamsRequest</a></td>
                <td><a href="#ibcgo.core.client.v1.QueryClientParamsResponse">QueryClientParamsResponse</a></td>
                <td><p>ClientParams queries all parameters of the ibc client.</p></td>
              </tr>
            
          </tbody>
        </table>

        
          
          
          <h4>Methods with HTTP bindings</h4>
          <table>
            <thead>
              <tr>
                <td>Method Name</td>
                <td>Method</td>
                <td>Pattern</td>
                <td>Body</td>
              </tr>
            </thead>
            <tbody>
            
              
              
              <tr>
                <td>ClientState</td>
                <td>GET</td>
                <td>/ibc/core/client/v1/client_states/{client_id}</td>
                <td></td>
              </tr>
              
            
              
              
              <tr>
                <td>ClientStates</td>
                <td>GET</td>
                <td>/ibc/core/client/v1/client_states</td>
                <td></td>
              </tr>
              
            
              
              
              <tr>
                <td>ConsensusState</td>
                <td>GET</td>
                <td>/ibc/core/client/v1/consensus_states/{client_id}/revision/{revision_number}/height/{revision_height}</td>
                <td></td>
              </tr>
              
            
              
              
              <tr>
                <td>ConsensusStates</td>
                <td>GET</td>
                <td>/ibc/core/client/v1/consensus_states/{client_id}</td>
                <td></td>
              </tr>
              
            
              
              
              <tr>
                <td>ClientParams</td>
                <td>GET</td>
                <td>/ibc/client/v1/params</td>
                <td></td>
              </tr>
              
            
            </tbody>
          </table>
          
        
    
      
      <div class="file-heading">
        <h2 id="ibcgo/core/client/v1/tx.proto">ibcgo/core/client/v1/tx.proto</h2><a href="#title">Top</a>
      </div>
      <p></p>

      
        <h3 id="ibcgo.core.client.v1.MsgCreateClient">MsgCreateClient</h3>
        <p>MsgCreateClient defines a message to create an IBC client</p>

        
          <table class="field-table">
            <thead>
              <tr><td>Field</td><td>Type</td><td>Label</td><td>Description</td></tr>
            </thead>
            <tbody>
              
                <tr>
                  <td>client_state</td>
                  <td><a href="#google.protobuf.Any">google.protobuf.Any</a></td>
                  <td></td>
                  <td><p>light client state </p></td>
                </tr>
              
                <tr>
                  <td>consensus_state</td>
                  <td><a href="#google.protobuf.Any">google.protobuf.Any</a></td>
                  <td></td>
                  <td><p>consensus state associated with the client that corresponds to a given
height. </p></td>
                </tr>
              
                <tr>
                  <td>signer</td>
                  <td><a href="#string">string</a></td>
                  <td></td>
                  <td><p>signer address </p></td>
                </tr>
              
            </tbody>
          </table>

          

        
      
        <h3 id="ibcgo.core.client.v1.MsgCreateClientResponse">MsgCreateClientResponse</h3>
        <p>MsgCreateClientResponse defines the Msg/CreateClient response type.</p>

        

        
      
        <h3 id="ibcgo.core.client.v1.MsgSubmitMisbehaviour">MsgSubmitMisbehaviour</h3>
        <p>MsgSubmitMisbehaviour defines an sdk.Msg type that submits Evidence for</p><p>light client misbehaviour.</p>

        
          <table class="field-table">
            <thead>
              <tr><td>Field</td><td>Type</td><td>Label</td><td>Description</td></tr>
            </thead>
            <tbody>
              
                <tr>
                  <td>client_id</td>
                  <td><a href="#string">string</a></td>
                  <td></td>
                  <td><p>client unique identifier </p></td>
                </tr>
              
                <tr>
                  <td>misbehaviour</td>
                  <td><a href="#google.protobuf.Any">google.protobuf.Any</a></td>
                  <td></td>
                  <td><p>misbehaviour used for freezing the light client </p></td>
                </tr>
              
                <tr>
                  <td>signer</td>
                  <td><a href="#string">string</a></td>
                  <td></td>
                  <td><p>signer address </p></td>
                </tr>
              
            </tbody>
          </table>

          

        
      
        <h3 id="ibcgo.core.client.v1.MsgSubmitMisbehaviourResponse">MsgSubmitMisbehaviourResponse</h3>
        <p>MsgSubmitMisbehaviourResponse defines the Msg/SubmitMisbehaviour response</p><p>type.</p>

        

        
      
        <h3 id="ibcgo.core.client.v1.MsgUpdateClient">MsgUpdateClient</h3>
        <p>MsgUpdateClient defines an sdk.Msg to update a IBC client state using</p><p>the given header.</p>

        
          <table class="field-table">
            <thead>
              <tr><td>Field</td><td>Type</td><td>Label</td><td>Description</td></tr>
            </thead>
            <tbody>
              
                <tr>
                  <td>client_id</td>
                  <td><a href="#string">string</a></td>
                  <td></td>
                  <td><p>client unique identifier </p></td>
                </tr>
              
                <tr>
                  <td>header</td>
                  <td><a href="#google.protobuf.Any">google.protobuf.Any</a></td>
                  <td></td>
                  <td><p>header to update the light client </p></td>
                </tr>
              
                <tr>
                  <td>signer</td>
                  <td><a href="#string">string</a></td>
                  <td></td>
                  <td><p>signer address </p></td>
                </tr>
              
            </tbody>
          </table>

          

        
      
        <h3 id="ibcgo.core.client.v1.MsgUpdateClientResponse">MsgUpdateClientResponse</h3>
        <p>MsgUpdateClientResponse defines the Msg/UpdateClient response type.</p>

        

        
      
        <h3 id="ibcgo.core.client.v1.MsgUpgradeClient">MsgUpgradeClient</h3>
        <p>MsgUpgradeClient defines an sdk.Msg to upgrade an IBC client to a new client</p><p>state</p>

        
          <table class="field-table">
            <thead>
              <tr><td>Field</td><td>Type</td><td>Label</td><td>Description</td></tr>
            </thead>
            <tbody>
              
                <tr>
                  <td>client_id</td>
                  <td><a href="#string">string</a></td>
                  <td></td>
                  <td><p>client unique identifier </p></td>
                </tr>
              
                <tr>
                  <td>client_state</td>
                  <td><a href="#google.protobuf.Any">google.protobuf.Any</a></td>
                  <td></td>
                  <td><p>upgraded client state </p></td>
                </tr>
              
                <tr>
                  <td>consensus_state</td>
                  <td><a href="#google.protobuf.Any">google.protobuf.Any</a></td>
                  <td></td>
                  <td><p>upgraded consensus state, only contains enough information to serve as a
basis of trust in update logic </p></td>
                </tr>
              
                <tr>
                  <td>proof_upgrade_client</td>
                  <td><a href="#bytes">bytes</a></td>
                  <td></td>
                  <td><p>proof that old chain committed to new client </p></td>
                </tr>
              
                <tr>
                  <td>proof_upgrade_consensus_state</td>
                  <td><a href="#bytes">bytes</a></td>
                  <td></td>
                  <td><p>proof that old chain committed to new consensus state </p></td>
                </tr>
              
                <tr>
                  <td>signer</td>
                  <td><a href="#string">string</a></td>
                  <td></td>
                  <td><p>signer address </p></td>
                </tr>
              
            </tbody>
          </table>

          

        
      
        <h3 id="ibcgo.core.client.v1.MsgUpgradeClientResponse">MsgUpgradeClientResponse</h3>
        <p>MsgUpgradeClientResponse defines the Msg/UpgradeClient response type.</p>

        

        
      

      

      

      
        <h3 id="ibcgo.core.client.v1.Msg">Msg</h3>
        <p>Msg defines the ibc/client Msg service.</p>
        <table class="enum-table">
          <thead>
            <tr><td>Method Name</td><td>Request Type</td><td>Response Type</td><td>Description</td></tr>
          </thead>
          <tbody>
            
              <tr>
                <td>CreateClient</td>
                <td><a href="#ibcgo.core.client.v1.MsgCreateClient">MsgCreateClient</a></td>
                <td><a href="#ibcgo.core.client.v1.MsgCreateClientResponse">MsgCreateClientResponse</a></td>
                <td><p>CreateClient defines a rpc handler method for MsgCreateClient.</p></td>
              </tr>
            
              <tr>
                <td>UpdateClient</td>
                <td><a href="#ibcgo.core.client.v1.MsgUpdateClient">MsgUpdateClient</a></td>
                <td><a href="#ibcgo.core.client.v1.MsgUpdateClientResponse">MsgUpdateClientResponse</a></td>
                <td><p>UpdateClient defines a rpc handler method for MsgUpdateClient.</p></td>
              </tr>
            
              <tr>
                <td>UpgradeClient</td>
                <td><a href="#ibcgo.core.client.v1.MsgUpgradeClient">MsgUpgradeClient</a></td>
                <td><a href="#ibcgo.core.client.v1.MsgUpgradeClientResponse">MsgUpgradeClientResponse</a></td>
                <td><p>UpgradeClient defines a rpc handler method for MsgUpgradeClient.</p></td>
              </tr>
            
              <tr>
                <td>SubmitMisbehaviour</td>
                <td><a href="#ibcgo.core.client.v1.MsgSubmitMisbehaviour">MsgSubmitMisbehaviour</a></td>
                <td><a href="#ibcgo.core.client.v1.MsgSubmitMisbehaviourResponse">MsgSubmitMisbehaviourResponse</a></td>
                <td><p>SubmitMisbehaviour defines a rpc handler method for MsgSubmitMisbehaviour.</p></td>
              </tr>
            
          </tbody>
        </table>

        
    

    <h2 id="scalar-value-types">Scalar Value Types</h2>
    <table class="scalar-value-types-table">
      <thead>
        <tr><td>.proto Type</td><td>Notes</td><td>C++</td><td>Java</td><td>Python</td><td>Go</td><td>C#</td><td>PHP</td><td>Ruby</td></tr>
      </thead>
      <tbody>
        
          <tr id="double">
            <td>double</td>
            <td></td>
            <td>double</td>
            <td>double</td>
            <td>float</td>
            <td>float64</td>
            <td>double</td>
            <td>float</td>
            <td>Float</td>
          </tr>
        
          <tr id="float">
            <td>float</td>
            <td></td>
            <td>float</td>
            <td>float</td>
            <td>float</td>
            <td>float32</td>
            <td>float</td>
            <td>float</td>
            <td>Float</td>
          </tr>
        
          <tr id="int32">
            <td>int32</td>
            <td>Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint32 instead.</td>
            <td>int32</td>
            <td>int</td>
            <td>int</td>
            <td>int32</td>
            <td>int</td>
            <td>integer</td>
            <td>Bignum or Fixnum (as required)</td>
          </tr>
        
          <tr id="int64">
            <td>int64</td>
            <td>Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint64 instead.</td>
            <td>int64</td>
            <td>long</td>
            <td>int/long</td>
            <td>int64</td>
            <td>long</td>
            <td>integer/string</td>
            <td>Bignum</td>
          </tr>
        
          <tr id="uint32">
            <td>uint32</td>
            <td>Uses variable-length encoding.</td>
            <td>uint32</td>
            <td>int</td>
            <td>int/long</td>
            <td>uint32</td>
            <td>uint</td>
            <td>integer</td>
            <td>Bignum or Fixnum (as required)</td>
          </tr>
        
          <tr id="uint64">
            <td>uint64</td>
            <td>Uses variable-length encoding.</td>
            <td>uint64</td>
            <td>long</td>
            <td>int/long</td>
            <td>uint64</td>
            <td>ulong</td>
            <td>integer/string</td>
            <td>Bignum or Fixnum (as required)</td>
          </tr>
        
          <tr id="sint32">
            <td>sint32</td>
            <td>Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int32s.</td>
            <td>int32</td>
            <td>int</td>
            <td>int</td>
            <td>int32</td>
            <td>int</td>
            <td>integer</td>
            <td>Bignum or Fixnum (as required)</td>
          </tr>
        
          <tr id="sint64">
            <td>sint64</td>
            <td>Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int64s.</td>
            <td>int64</td>
            <td>long</td>
            <td>int/long</td>
            <td>int64</td>
            <td>long</td>
            <td>integer/string</td>
            <td>Bignum</td>
          </tr>
        
          <tr id="fixed32">
            <td>fixed32</td>
            <td>Always four bytes. More efficient than uint32 if values are often greater than 2^28.</td>
            <td>uint32</td>
            <td>int</td>
            <td>int</td>
            <td>uint32</td>
            <td>uint</td>
            <td>integer</td>
            <td>Bignum or Fixnum (as required)</td>
          </tr>
        
          <tr id="fixed64">
            <td>fixed64</td>
            <td>Always eight bytes. More efficient than uint64 if values are often greater than 2^56.</td>
            <td>uint64</td>
            <td>long</td>
            <td>int/long</td>
            <td>uint64</td>
            <td>ulong</td>
            <td>integer/string</td>
            <td>Bignum</td>
          </tr>
        
          <tr id="sfixed32">
            <td>sfixed32</td>
            <td>Always four bytes.</td>
            <td>int32</td>
            <td>int</td>
            <td>int</td>
            <td>int32</td>
            <td>int</td>
            <td>integer</td>
            <td>Bignum or Fixnum (as required)</td>
          </tr>
        
          <tr id="sfixed64">
            <td>sfixed64</td>
            <td>Always eight bytes.</td>
            <td>int64</td>
            <td>long</td>
            <td>int/long</td>
            <td>int64</td>
            <td>long</td>
            <td>integer/string</td>
            <td>Bignum</td>
          </tr>
        
          <tr id="bool">
            <td>bool</td>
            <td></td>
            <td>bool</td>
            <td>boolean</td>
            <td>boolean</td>
            <td>bool</td>
            <td>bool</td>
            <td>boolean</td>
            <td>TrueClass/FalseClass</td>
          </tr>
        
          <tr id="string">
            <td>string</td>
            <td>A string must always contain UTF-8 encoded or 7-bit ASCII text.</td>
            <td>string</td>
            <td>String</td>
            <td>str/unicode</td>
            <td>string</td>
            <td>string</td>
            <td>string</td>
            <td>String (UTF-8)</td>
          </tr>
        
          <tr id="bytes">
            <td>bytes</td>
            <td>May contain any arbitrary sequence of bytes.</td>
            <td>string</td>
            <td>ByteString</td>
            <td>str</td>
            <td>[]byte</td>
            <td>ByteString</td>
            <td>string</td>
            <td>String (ASCII-8BIT)</td>
          </tr>
        
      </tbody>
    </table>
  </body>
</html>

