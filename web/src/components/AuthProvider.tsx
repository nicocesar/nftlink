import React, {useState, SetStateAction} from "react";

const AuthContext = React.createContext({
  token: "",
  getConnectedWalletAddress: () => {},
  updateWalletAddress: (value:string) => {},
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

interface ProtectedRouteProps {
  children: JSX.Element | null;
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

    const updateWalletAddress = (value:string) =>{
        setCurrentAccount(value);
    }

    const getConnectedWalletAddress = () => {
        return currentAccount;
    }

    const value = {
        token,
        getConnectedWalletAddress,
        updateWalletAddress,
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