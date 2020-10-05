
# go-scdo
[![Build Status](https://travis-ci.org/scdo/go-scdo.svg?branch=master)](https://travis-ci.org/scdo/go-scdo)

|        Features        |      Descriptions                                                                              |
|:-----------------------|------------------------------------------------------------------------------------------------|
| **Sharding**           | 4 shards, transactions within the same shard and between different shards are supported<br/> higher transaction fee for cross-shard transaction                                  |
| **Smart Contracts**    | smart contracts are supported within the same shard                                          |
| **Scdo Wallet**       | easy-to-use wallet                                                                             |
| **High TPS**           | same shard TPS: 250/shard, cross shard TPS: 6/shard                                           |
| **Auditable Supply**   | total supply: 300,000,000 Scdos, all from mining                              |
| **Consensus Algorithm**| ZPOW algorithm                                                |
| **Mining Reward**      | 3150000 blocks/era and block reward at each era follows [6, 4, 3, 2.5, 2, 2, 1.5, 1.5] order until reaches the last reward of 1.5 Scdos |
| **Transaction Fee**    | self-customized transaction fee, higher fee for cross-shard transaction                        |
| **Block**              | 100 KB block size, 20 seconds block time, ~6000 transactions per block                         |


The official Golang implementation of Scdo. Scdo is an open source blockchain project which consists of advanced sharding technology, innovative ZPoW consensus algorithm and scalable subchain protocol. [https://scdo.pro](https://scdo.pro)

The current mainnet release: Scdo mainchain is powered by a new anti-ASIC consensus PoW algorithm, which requires scientific calculation related to randomized matrix. The mainchain has four shards. Users can perform transactions within a shard or across shards. However, currently smart contracts can only be executed within the same shard. Scdo subchains are under development. 

# Download (without building)
If you want to run the node directly and use client without setting up the compiling enviroment and building the executable files, you can choose the right version to download and run:

| Operation System |      Download Link     |
|---------|----------------------------------------------------------|
| Linux   | [https://github.com/scdoproject/go-scdo/releases]|
| MacOs   | [https://github.com/scdoproject/go-scdo/releases]|
| Windows | [https://github.com/scdoproject/go-scdo/releases]|

# Or Download & Build the source

- Building the Scdo project requires both a Go (version 1.12.7 ONLY at this moment) compiler, Git, and a C compiler.

- Clone the go-scdo repository to the GOPATH directory:

```
go get -u -v github.com/scdoproject/go-scdo/...
```

- Once successfully cloned source code:

```
cd GOPATH/src/github.com/scdoproject/go-scdo/
```

- Linux & Mac amd64

```
make all
```

- Windows amd64

```
buildall.bat
```

# Run Scdo
A simple version Scdo mining tutorial: English-[Scdo MiningTutorial](https://github.com/scdoproject/go-scdo/releases/tag/v1.0.1-MiningTutorial_Eng), 中文-[Scdo 挖矿教程中文简版](https://github.com/scdoproject/go-scdo/releases/tag/v1.0.1-%E4%B8%AD%E6%96%87%E7%AE%80%E7%89%88%E6%8C%96%E7%9F%BF%E6%95%99%E7%A8%8B).

For running a node, please refer to [Get Started](https://scdotech.gitbook.io/wiki/developer/go-scdo/gettingstarted)([Older version](https://scdoteam.github.io/scdo-doc/docs/Getting-Started-With-Seele Credo.html)).
For more usage details and deeper explanations, please consult the [Scdo Wiki](https://scdotech.gitbook.io/wiki/)([Older version](https://scdoteam.github.io/scdo-doc/index.html)).

# Contribution

Thank you for considering helping out with our source code. We appreciate any contributions, even the smallest fixes.

Here are some guidelines before you start:
* Code must adhere to the official Go [formatting](https://golang.org/doc/effective_go.html#formatting) guidelines (i.e. uses [gofmt](https://golang.org/cmd/gofmt/)).
* Pull requests need to be based on and opened against the `master` branch.
* We use reviewable.io as our review tool for any pull request. Please submit and follow up on your comments in this tool. After you submit a PR, there will be a `Reviewable` button in your PR. Click this button, it will take you to the review page (it may ask you to login).
* If you have any questions, feel free to join [chat room](https://gitter.im/scdoteamchat/dev) to communicate with our core team.

# Resources

* [Scdo Website](https://scdo.pro/)
* [Telegram Group](https://t.me/scdotech)
* [Roadmap](https://scdo.pro/)
* [Scdo Wiki](https://scdotech.gitbook.io/wiki/)
* [scdo-sdk-javascript](https://www.npmjs.com/package/scdo-sdk-javascript)

# License

[go-scdo/LICENSE](https://github.com/scdoproject/go-scdo/blob/master/LICENSE)
