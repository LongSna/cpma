SET config.toml BEFORE USE.  
usage:  
`RPC_URL` - rpc url  
`ContractAddresses`- contract addresses you want to filter  
`FromBlock/ToBlock` - scan block range  
`Step` - step depends on your rpc config  
Example:  
filter the addresses that have purchased   
`0x1BB9b64927e0C5e207C9DB4093b3738Eef5D8447` and 
`0x89d584A1EDB3A70B3B07963F9A3eA5399E38b136`   
in the range of `FromBlock = 19096064` 
`ToBlock = 19096066`  
`step= 100`

RUNNING screenshot:  
<img width="661" alt="image" src="https://github.com/LongSna/cpma/assets/121377806/91367fac-06d0-48f4-a130-78a03c45151f">  

output result:`0x45da98984b745a6ba1f4a67c842e63664d7d55f1`  

This address corresponds to the two hashes(Under normal circumstances, there should be multiple addresses here.） 
`0xe946798b8f1547ae0221196ece7b17c9a2e8dbcb48933dd4396d537102219d3b`and`0xc26de61580d510d6eedc96eae97c590157c841b8c4e11c2bb573b32a217ebad2`



