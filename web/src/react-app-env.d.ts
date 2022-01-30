/// <reference types="react-scripts" />

// https://ethereum.stackexchange.com/questions/94439/trying-to-use-window-ethereum-request-in-typescript-errors-out-with-property-re/94468
interface Window {
    ethereum: any;
  }