package server

const backendErrorTmpl = `<html>
  <head>
    <title>Box not available &middot; Nitrous.IO</title>
    <link rel="stylesheet" href="https://fonts.googleapis.com/css?family=Open+Sans:400,300,700&subset=latin">

    <style type="text/css">
      body {
        padding-top: 20px;
        background: #2A323A;
      }

      .container {
        margin-right: auto;
        margin-left: auto;
        width: 940px;
      }

      #ft-header {
        padding-left: 8px;
        border-bottom: 2px solid  #212830;
      }

      #ft-body {
        padding: 48px 0;
        text-align: center;
        color: white;
        font-family: "Open Sans", "Helvetica Neue", Helvetica, Arial, sans-serif;
      }

      .row {
        margin-left: -20px;
        padding-bottom: 10px;
      }

      .row::before,
      .row::after {
        display: table;
        content: "";
      }

      .row::after {
        clear: both;
      }

      .span3 {
        width: 220px;
        float: left;
        margin-left: 20px;
      }

      h1, h2, h3, h4, h5, h6 {
        font-family: "Open Sans", "Helvetica Neue", Helvetica, Arial, sans-serif;
        font-weight: normal;
        color: white;
        margin: 0;
        text-rendering: optimizelegibility;
      }

      h1.logo {
        margin-left: 32px;
        font-size: 18px;
        line-height: 22px;
      }

      img.logo {
        max-width: 100%;
        vertical-align: middle;
        border: 0;

        float: left;
        -webkit-border-radius: 1px;
        -moz-border-radius: 1px;
        border-radius: 1px;
      }

      h1.logo a {
        color:  white;
        text-shadow: none;
        position: relative;
        -webkit-mask: -webkit-gradient(radial, 17 17, 122, 17 17, 137, from(black), color-stop(0.5, rgba(0, 0, 0, 0.199)), to(black));
        text-decoration: none;
      }

      a {
        color: #08c;
      }
    </style>

  </head>

  <body>
    <div class="container" id="ft-header">
      <div class="row">
        <div class="span3">
          <img alt="Nitrous.IO" class="logo" src="https://www.nitrous.io/images/simple-logo.png" width="24">
          <h1 class="logo"><a href="/">Nitrous.IO</a></h1>
        </div>
      </div>
    </div>

    <div class="container" id="ft-body">
      timeout connecting to backend
      <h3>The box {{.BoxName}} you're trying to preview is not available.</h3>
      <p>{{.Error}}</p>
      <p>Please refer to this <a href="http://help.nitrous.io/preview/">support article</a> for more details.</p>
    </div>
  </body>
</html>
`
