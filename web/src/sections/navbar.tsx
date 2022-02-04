import { ReactNode } from 'react';

type INavbarProps = {
  children: ReactNode;
};

const NavbarTwoColumns = (props: INavbarProps) => (
    <div className="flex flex-wrap justify-between items-center">
        <nav>
            <ul className="navbar flex items-center font-medium text-xl text-gray-800">
                {props.children}
            </ul>
    </nav>
  </div>
);

export { NavbarTwoColumns };