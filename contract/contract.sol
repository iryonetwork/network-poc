pragma solidity ^0.4.0;
contract PoC {
    mapping (address => mapping (address => bool)) connected;

    /* connects the msg.sender and to address as connected  */
    function grantAccess(address _to) public returns (bool success) {
        connected[msg.sender][_to] =true;          // Set the connection
        return true;
    }

    /* connects the msg.sender and to address as connected  */
    function revokeAccess(address _from) public returns (bool success) {
        connected[msg.sender][_from] =false;          // Remove the connection
        return true;
    }

    /* Get the amount of remaining tokens to spend */
    function accessGranted(address _from, address _to) public constant returns (bool is_connected) {
        return connected[_from][_to];
    }
}
