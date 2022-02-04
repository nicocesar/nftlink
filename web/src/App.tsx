import React from "react";
import { useRoutes } from 'react-router-dom';
import Home from "./containers/Home";
import Metaverse from './containers/Metaverse'
import NoMatch from './components/NoMatch'

declare global {
    interface Window {
        ethereum:any;
    }
}

let FB = window.ethereum;

const App: React.FC = (): JSX.Element => {

  const mainRoutes = [
  { path: '/', element: <Home />},
  { path: '/metaverse', element: <Metaverse />},
  { path: '/*', element: <NoMatch />},
  ];

  const routing = useRoutes(mainRoutes);

  return <>{routing}</>;
}

export default App;
