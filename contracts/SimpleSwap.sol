// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";

contract SimpleSwap {
    IERC20 public token;
    uint256 public rate = 1000; // 1 ETH = 1000 MTK

    constructor(address tokenAddress) {
        token = IERC20(tokenAddress);
    }

    // 用户用ETH换Token
    function buyToken() public payable {
        uint256 amount = msg.value * rate;
        require(token.balanceOf(address(this)) >= amount, "Insufficient token in contract");
        token.transfer(msg.sender, amount);
    }

    // 用户用Token换ETH
    function sellToken(uint256 tokenAmount) public {
        uint256 ethAmount = tokenAmount / rate;
        require(address(this).balance >= ethAmount, "Insufficient ETH in contract");
        token.transferFrom(msg.sender, address(this), tokenAmount);
        payable(msg.sender).transfer(ethAmount);
    }

    // 合约owner充值Token
    function depositToken(uint256 amount) public {
        token.transferFrom(msg.sender, address(this), amount);
    }

    // 合约owner提取ETH
    function withdrawETH(uint256 amount) public {
        payable(msg.sender).transfer(amount);
    }

    receive() external payable {}
} 