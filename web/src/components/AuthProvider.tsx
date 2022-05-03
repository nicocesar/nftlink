import React, { useState } from "react";

const AuthContext = React.createContext({
  token: "",
  connectWallet: (event: React.MouseEvent<HTMLButtonElement, MouseEvent>) => {},
  onGetConnectedWalletAddress: () => {},
  onUpdateWalletAddress: (value:string) => {},
  onLogin: () => { },
  onLogout: () => { },
});

const fakeAuth = () =>
    new Promise<string>((resolve) => {
        setTimeout(() => resolve('2342f2f1d131rf12'), 250);
    });

interface Props {
  children?: JSX.Element | JSX.Element[];
}

const useAuth = () => {
  return React.useContext(AuthContext);
};

const AuthProvider = ({ children }: Props) => {
    const [token, setToken] = useState<string>("")
    const [currentAccount, setCurrentAccount] = useState("");

    const handleLogin = async () => {
        const token = await fakeAuth();
        setToken(token);
    };

    const handleLogout = () => {
        setToken("");
    };

    const updateWalletAddress = async (value:string) =>{
        setCurrentAccount(value);
    }

    const getConnectedWalletAddress = async () => {
        return currentAccount;
    }

    const connectWallet = async (event: React.MouseEvent<HTMLButtonElement>) => {
        try {
          const { ethereum } = window;

          if (!ethereum) {
            alert("Get MetaMask!");
            return;
          }

          const accounts = await ethereum.request({ method: "eth_requestAccounts" });

          console.log("Connected", accounts[0]);
          updateWalletAddress(accounts[0]);
        } catch (error) {
          console.log(error)
        }
      }

    const value = {
        token,
        connectWallet,
        onGetConnectedWalletAddress : getConnectedWalletAddress,
        onUpdateWalletAddress : updateWalletAddress,
        onLogin: handleLogin,
        onLogout: handleLogout,
    };



    return (
        <AuthContext.Provider value={value}>
            {children}
        </AuthContext.Provider>
    );

}

export {AuthProvider, useAuth};